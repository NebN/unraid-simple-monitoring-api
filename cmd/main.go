package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/NebN/unraid-simple-monitoring-api/internal/monitor"
	"gopkg.in/yaml.v3"
)

const PORT = "24940"

func main() {
	mux := http.NewServeMux()
	confPath := os.Getenv("CONF_PATH")
	conf, err := readConf(confPath)
	if err != nil {
		slog.Error("Cannot read conf.yml", slog.String("error", err.Error()))
		return
	} else {
		slog.Debug("Configuration", "conf", conf)
	}

	rootHandler := NewHandler(conf)
	mux.Handle("/", &rootHandler)

	slog.Info(fmt.Sprintf("API running on port %s ...", PORT))
	err = http.ListenAndServe(fmt.Sprintf(":%s", "24940"), mux)
	if err != nil {
		slog.Error("Cannot start API", slog.String("error", err.Error()))
	}
}

type Conf struct {
	Networks []string `yaml:"networks"`
	Disks    struct {
		Cache []string `yaml:"cache"`
		Array []string `yaml:"array"`
	} `yaml:"disks"`
}

func readConf(path string) (Conf, error) {
	conf := Conf{}
	content, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(content, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

type handler struct {
	NetworkMonitor monitor.NetworkMonitor
	DiskMonitor    monitor.DiskMonitor
	CpuMonitor     monitor.CpuMonitor
}

func NewHandler(conf Conf) (handler handler) {
	handler.DiskMonitor = monitor.NewDiskMonitor(conf.Disks.Cache, conf.Disks.Array)
	handler.NetworkMonitor = monitor.NewNetworkMonitor(conf.Networks)
	handler.CpuMonitor = monitor.NewCpuMonitor()
	return
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	cache, array := h.DiskMonitor.ComputeDiskUsage()
	network := h.NetworkMonitor.ComputeNetworkRate()
	cacheTotal := monitor.AggregateDiskStatuses(cache)
	arrayTotal := monitor.AggregateDiskStatuses(array)
	networkTotal := monitor.AggregateNetworkRates(network)
	cpu := h.CpuMonitor.ComputeCpuStatus()

	response := Report{
		Cache:        cache,
		Array:        array,
		Network:      network,
		ArrayTotal:   arrayTotal,
		CacheTotal:   cacheTotal,
		NetworkTotal: networkTotal,
		Cpu:          cpu,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error trying to respond to API call", slog.String("error", err.Error()))
		w.Write([]byte(err.Error()))
	} else {
		w.Write([]byte(responseJson))
	}
}

type Report struct {
	Array        []monitor.DiskStatus  `json:"array"`
	Cache        []monitor.DiskStatus  `json:"cache"`
	Network      []monitor.NetworkRate `json:"network"`
	ArrayTotal   monitor.DiskStatus    `json:"array_total"`
	CacheTotal   monitor.DiskStatus    `json:"cache_total"`
	NetworkTotal monitor.NetworkRate   `json:"network_total"`
	Cpu          monitor.CpuStatus     `json:"cpu"`
}
