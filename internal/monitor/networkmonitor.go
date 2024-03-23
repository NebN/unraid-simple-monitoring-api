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
}

func NewNetworkMonitor(inames []string) (monitor NetworkMonitor) {
	snapshots := make([]NetworkSnapshot, 0, len(inames))
	for _, iname := range inames {
		snapshots = append(snapshots, newNetworkSnapshot(iname))
	}
	monitor.snapshots = snapshots
	return
}

func (monitor *NetworkMonitor) ComputeNetworkRate() []NetworkRate {
	monitor.mu.Lock()
	defer monitor.mu.Unlock()

	var wg sync.WaitGroup
	snapshotChan := make(chan NetworkSnapshot, len(monitor.snapshots))
	rateChan := make(chan NetworkRate, len(monitor.snapshots))

	for _, snapshot := range monitor.snapshots {
		wg.Add(1)
		go networkRate(snapshot, &wg, snapshotChan, rateChan)
	}

	wg.Wait()
	close(snapshotChan)
	close(rateChan)

	rates := make([]NetworkRate, 0, len(monitor.snapshots))
	snapshots := make([]NetworkSnapshot, 0, len(monitor.snapshots))

	for snapshot := range snapshotChan {
		snapshots = append(snapshots, snapshot)
	}
	for rate := range rateChan {
		rates = append(rates, rate)
	}

	monitor.snapshots = snapshots

	return rates
}

func networkRate(previousSnapshot NetworkSnapshot, wg *sync.WaitGroup, snapshotChan chan NetworkSnapshot, rateChan chan NetworkRate) {
	defer wg.Done()

	snapshot := newNetworkSnapshot(previousSnapshot.Iname)

	ratePerSecond := func(t0Reading uint64, t1Reading uint64, t0 time.Time, t1 time.Time) (float64, float64) {
		readingDiff := (t1Reading - t0Reading)
		rate := float64(readingDiff) / t1.Sub(t0).Seconds()
		rateMebiBytes := util.BytesToMebiBytes(rate)
		rateMegaBits := util.BytesToBits(util.BytesToMegaBytes(rate))

		return rateMebiBytes, rateMegaBits
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

	snapshotChan <- snapshot
	rateChan <- rate
}

func newNetworkSnapshot(iname string) (network NetworkSnapshot) {

	usageInBps := func(direction string, c chan uint64, ts chan time.Time) {

		defer close(c)
		defer close(ts)

		now := time.Now()
		res, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/%s_bytes", iname, direction))
		if err != nil {
			slog.Error("Cannot read network data", "interface", iname, slog.String("error", err.Error()))
			return
		}

		stringBytes := strings.TrimSuffix(string(res), "\n")

		uint64Bytes, err := strconv.ParseUint(stringBytes, 10, 64)
		if err != nil {
			fmt.Println("error!", err)
			return
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
	network.Iname = iname
	return
}

func AggregateNetworkRates(networks []NetworkRate) (status NetworkRate) {
	for _, network := range networks {
		status.RxMbps = status.RxMbps + network.RxMbps
		status.TxMbps = status.TxMbps + network.TxMbps
		status.RxMiBs = status.RxMiBs + network.RxMiBs
		status.TxMiBs = status.TxMiBs + network.TxMiBs
	}
	status.Iname = "total"
	return
}
