package monitor

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NebN/unraid-simple-monitoring-api/internal/util"
)

type NetworkRate struct {
	Iname  string  `json:"interface"`
	RxMiBs float64 `json:"rx_MiBs"`
	TxMiBs float64 `json:"tx_MiBs"`
	RxMbps float64 `json:"rx_Mbps"`
	TxMbps float64 `json:"tx_Mbps"`
}

type NetworkSnapshot struct {
	Iname string
	Rx    uint64
	Tx    uint64
	RxTs  time.Time
	TxTs  time.Time
}

type NetworkMonitor struct {
	snapshots []NetworkSnapshot
	mu        sync.Mutex
	fsPrefix  string
}

func NewNetworkMonitor(inames []string) (monitor NetworkMonitor) {
	var hostFsPrefix, isSet = os.LookupEnv("HOSTFS_PREFIX")
	if isSet {
		slog.Debug("Network host prefix is set", "value", hostFsPrefix)
		monitor.fsPrefix = hostFsPrefix
	}

	snapshots := make([]NetworkSnapshot, len(inames))
	for i, iname := range inames {
		snapshots[i] = newNetworkSnapshot(iname, monitor.fsPrefix)
	}
	monitor.snapshots = snapshots
	return
}

func (monitor *NetworkMonitor) ComputeNetworkRate() []NetworkRate {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	var wg sync.WaitGroup
	snapshotChan := make(chan util.IndexedValue[NetworkSnapshot], len(monitor.snapshots))
	rateChan := make(chan util.IndexedValue[NetworkRate], len(monitor.snapshots))

	for i, snapshot := range monitor.snapshots {
		wg.Add(1)
		go networkRate(i, monitor.fsPrefix, snapshot, &wg, snapshotChan, rateChan)
	}

	wg.Wait()
	close(snapshotChan)
	close(rateChan)

	rates := make([]NetworkRate, len(monitor.snapshots))
	snapshots := make([]NetworkSnapshot, len(monitor.snapshots))

	for snapshot := range snapshotChan {
		snapshots[snapshot.Index] = snapshot.Value
	}
	for rate := range rateChan {
		rates[rate.Index] = rate.Value
	}

	monitor.snapshots = snapshots

	return rates
}

func networkRate(
	index int,
	fsPrefix string,
	previousSnapshot NetworkSnapshot,
	wg *sync.WaitGroup,
	snapshotChan chan util.IndexedValue[NetworkSnapshot],
	rateChan chan util.IndexedValue[NetworkRate]) {

	defer wg.Done()

	snapshot := newNetworkSnapshot(previousSnapshot.Iname, fsPrefix)

	ratePerSecond := func(t0Reading uint64, t1Reading uint64, t0 time.Time, t1 time.Time) (float64, float64) {
		readingDiff := (t1Reading - t0Reading)
		deltaTime := t1.Sub(t0).Seconds()
		if deltaTime > 0 {
			rate := float64(readingDiff) / deltaTime
			rateMebiBytes := util.BytesToMebiBytes(rate)
			rateMegaBits := util.BytesToBits(util.BytesToMegaBytes(rate))
			slog.Debug("Network computing rate per second",
				"interface", snapshot.Iname,
				"t0_value", t0Reading,
				"t0", t0,
				"t1_value", t1Reading,
				"t1", t1)
			return rateMebiBytes, rateMegaBits
		} else {
			slog.Warn("Network delta time between snapshots is 0, rate will be returned as 0", slog.String("interface", previousSnapshot.Iname))
		}
		return 0, 0
	}

	rxMiBs, rxMbps := ratePerSecond(previousSnapshot.Rx, snapshot.Rx, previousSnapshot.RxTs, snapshot.RxTs)
	txMiBs, txMbps := ratePerSecond(previousSnapshot.Tx, snapshot.Tx, previousSnapshot.TxTs, snapshot.TxTs)

	rate := NetworkRate{
		RxMiBs: util.RoundTwoDecimals(rxMiBs),
		TxMiBs: util.RoundTwoDecimals(txMiBs),
		RxMbps: util.RoundTwoDecimals(rxMbps),
		TxMbps: util.RoundTwoDecimals(txMbps),
		Iname:  previousSnapshot.Iname,
	}

	slog.Debug("Network", "rate", rate)
	snapshotChan <- util.IndexedValue[NetworkSnapshot]{Index: index, Value: snapshot}
	rateChan <- util.IndexedValue[NetworkRate]{Index: index, Value: rate}
}

func newNetworkSnapshot(iname string, fsPrefix string) (network NetworkSnapshot) {
	network.Iname = iname

	usageInBps := func(direction string, c chan uint64, ts chan time.Time) {

		defer close(c)
		defer close(ts)

		now := time.Now()
		res, err := os.ReadFile(fmt.Sprintf("%s/sys/class/net/%s/statistics/%s_bytes", fsPrefix, iname, direction))
		if err != nil {
			slog.Error("Network cannot read data", "interface", iname, slog.String("error", err.Error()))
			return
		}

		stringBytes := strings.TrimSuffix(string(res), "\n")

		uint64Bytes, err := strconv.ParseUint(stringBytes, 10, 64)
		if err != nil {
			slog.Error("Network cannot parse data from /sys/class/net/",
				slog.String("trying to parse", stringBytes),
				slog.String("error", err.Error()))
		}

		c <- uint64Bytes
		ts <- now
	}

	rxChan := make(chan uint64)
	txChan := make(chan uint64)
	rxTsChan := make(chan time.Time)
	txTsChan := make(chan time.Time)

	go usageInBps("rx", rxChan, rxTsChan)
	go usageInBps("tx", txChan, txTsChan)

	rx, ok := <-rxChan
	if !ok {
		rx = 0
	}

	tx, ok := <-txChan
	if !ok {
		tx = 0
	}

	rxTs, ok := <-rxTsChan
	if !ok {
		rxTs = time.Now()
	}

	txTs, ok := <-txTsChan
	if !ok {
		txTs = time.Now()
	}

	network.Rx = rx
	network.Tx = tx
	network.RxTs = rxTs
	network.TxTs = txTs

	slog.Debug("Network", "snapshot", network)
	return
}

func AggregateNetworkRates(networks []NetworkRate) (status NetworkRate) {
	names := make([]string, 0, len(networks))

	for _, network := range networks {
		names = append(names, network.Iname)
		status.RxMbps = status.RxMbps + network.RxMbps
		status.TxMbps = status.TxMbps + network.TxMbps
		status.RxMiBs = status.RxMiBs + network.RxMiBs
		status.TxMiBs = status.TxMiBs + network.TxMiBs
		slog.Debug("Network aggregation", "network", network, "running_total", status)
	}

	status.Iname = strings.Join(names, " ")

	return
}
