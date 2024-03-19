package main

import (
	"fmt"
	"net/http"
	"syscall"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
	"sync"
	"math"
)


func main() {
    mux := http.NewServeMux()
    rootHandler := NewHandler()

    mux.Handle("/", &rootHandler) 

    fmt.Println("API running...")
	http.ListenAndServe(":24940", mux)
}

type handler struct {
	NetworkMonitor NetworkMonitor
}

func NewHandler() (handler handler) {
	handler.NetworkMonitor = NewNetworkMonitor("eth0")
	return
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	disk, _ := DiskUsage("/mnt/cache")
	fmt.Println("disk", disk)

	network := h.NetworkMonitor.ComputeNetworkRate()
	fmt.Println("network", network)

	response := Report{Cache: disk, Network: network}
	fmt.Println("response", response)

	responseJson, err := json.Marshal(response)
	if err != nil {
		fmt.Println("error!", err)
	}

	fmt.Println("responseJson", responseJson)

    w.Write([]byte(responseJson))
}

type Report struct {
//	array DiskStatus
	Cache DiskStatus `json:"cache"`
	Network NetworkRate`json:"network"`
}

type DiskStatus struct {
	Path string `json:"mount"`
    Total uint64 `json:"total"`
    Used uint64 `json:"used"`
    Free uint64 `json:"free"` 
	FreePercent float64 `json:"free_percent"` 
	UsedPercent float64 `json:"used_percent"` 
}

func DiskUsage(path string) (DiskStatus, bool) {
    fs := syscall.Statfs_t{}
    err := syscall.Statfs(path, &fs)
    if err != nil {
    	return DiskStatus{}, false
    }

	total := bytesToGibiBytes(fs.Blocks * uint64(fs.Bsize))
	free := bytesToGibiBytes(fs.Bfree * uint64(fs.Bsize))
	used := total - free
	freePercent := roundTwoDecimals(float64(free) / float64(total) * 100)
	usedPercent := roundTwoDecimals(100 - freePercent)
	return DiskStatus{Path: path, Total: total, Free: free, Used: used, FreePercent: freePercent, UsedPercent: usedPercent}, true
}

type NetworkSnapshot struct {
    Rx uint64 
    Tx uint64
    RxTs time.Time 
    TxTs time.Time
}

type NetworkRate struct {
    Iname string `json:"interface"`
	RxMiBs float64 `json:"rx_MiBs"`
    TxMiBs float64 `json:"tx_MiBs"`
}

type NetworkMonitor struct {
	Iname string
	snapshot NetworkSnapshot
}

func NewNetworkMonitor(iname string) (monitor NetworkMonitor) {
	monitor.Iname = iname	
	monitor.snapshot = newNetworkSnapshot(iname)
	return
}

func (monitor *NetworkMonitor) ComputeNetworkRate() (rate NetworkRate) {
	snapshot := newNetworkSnapshot(monitor.Iname)
	
	differencePerSecond := func (t0Reading uint64, t1Reading uint64, t0 time.Time, t1 time.Time) float64 {
		readingDiff := (t1Reading - t0Reading)
		difference := float64(readingDiff) / t1.Sub(t0).Seconds()
		differenceMebi := bytesToMebiBytes(difference)

		return differenceMebi
	}

	previousSnapshot := monitor.snapshot
	rxBps := differencePerSecond(previousSnapshot.Rx, snapshot.Rx, previousSnapshot.RxTs, snapshot.RxTs)
	txBps := differencePerSecond(previousSnapshot.Tx, snapshot.Tx, previousSnapshot.TxTs, snapshot.TxTs)

	rxMiBs := rxBps 
	txMiBs := txBps

	rate.RxMiBs = roundTwoDecimals(rxMiBs)
	rate.TxMiBs = roundTwoDecimals(txMiBs)
	rate.Iname = monitor.Iname

	monitor.snapshot = snapshot

	return
}

func newNetworkSnapshot(iname string) (network NetworkSnapshot) {
   
	var wg sync.WaitGroup
	wg.Add(2)

    usageInBps := func (direction string, c chan uint64) {
		
		defer wg.Done()
        
		res, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/%s_bytes", iname, direction))
		if err != nil {
			fmt.Println("error!", err)
			return
		}

		stringBytes := strings.TrimSuffix(string(res), "\n")
		
		uint64Bytes, err := strconv.ParseUint(stringBytes, 10, 64)
		if err != nil {
			fmt.Println("error!", err)
			return
		}
		
		c <- uint64Bytes
    }

	rx_chan := make(chan uint64)
	tx_chan := make(chan uint64)

	t0 := time.Now()
	go usageInBps("rx", rx_chan)
	go usageInBps("tx", tx_chan)

	rx, ok := <- rx_chan
	if !ok {
		rx = 0
	}

	tx, ok := <- tx_chan
	if !ok {
		tx = 0
	}
	
	wg.Wait()

    network.Rx = rx
    network.Tx = tx
	network.RxTs = t0
	network.TxTs = t0
    return
}

func roundTwoDecimals(n float64) float64 {
	return math.Round(n*100)/100
}

func bytesToMebiBytes(b float64) float64 {
	mantissa, exponent := math.Frexp(b)
	return math.Ldexp(mantissa, exponent - 20)
}

func bytesToGibiBytes(b uint64) uint64 {
	return b >> 30 
}

