package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/NebN/unraid-simple-monitoring-api/internal/monitor"
	"gopkg.in/yaml.v3"
)

const PORT = "24940"

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	mux := http.NewServeMux()
	confPath := os.Getenv("CONF_PATH")
	conf, err := readConf(confPath)
	if err != nil {
		switch err := err.(type) {
		case *os.PathError:
			defaultInfo := "Default location is /mnt/user/appdata/unraid-simple-monitoring-api/conf.yml. " +
				"More info @ https://github.com/NebN/unraid-simple-monitoring-api"

			if strings.Contains(err.Error(), "is a directory") {
				slog.Error("Configuration file has been created as a directory. " +
					"Please delete it and create a configuration file in its place. " + defaultInfo)
			} else {
				slog.Error("Configuration file not found. Please create it. " + defaultInfo)
			}

		case *yaml.TypeError:
			slog.Error("Configuration file is malformed", "error", err.Error())

		default:
			slog.Error("Unable to read configuration file", "error", err.Error(), "type", reflect.TypeOf(err))
		}
		return
	} else {
		var loggingLevel slog.Level
		loggingLevel.UnmarshalText([]byte(conf.LoggingLevel))
		slog.SetLogLoggerLevel(loggingLevel)
		slog.Info("Logging", slog.String("level", loggingLevel.Level().String()))
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
	LoggingLevel string   `yaml:"loggingLevel"`
	CpuTemp      *string  `yaml:"cpuTemp"`
	Include      []string `yaml:"include"`
	Exclude      []string `yaml:"exclude"`
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
	MemoryMonitor  monitor.MemoryMonitor
}

func NewHandler(conf Conf) (handler handler) {
	handler.DiskMonitor = monitor.NewDiskMonitor(conf.Disks.Cache, conf.Disks.Array)
	handler.NetworkMonitor = monitor.NewNetworkMonitor(conf.Networks)
	handler.CpuMonitor = monitor.NewCpuMonitor(conf.CpuTemp)
	return
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	slog.Debug("Request received", slog.String("request", fmt.Sprintf("%+v\n", r)))

	cache, array, parity := h.DiskMonitor.ComputeDiskUsage()
	network := h.NetworkMonitor.ComputeNetworkRate()
	cacheTotal := monitor.AggregateDiskStatuses(cache)
	arrayTotal := monitor.AggregateDiskStatuses(array)
	networkTotal := monitor.AggregateNetworkRates(network)
	cpu := h.CpuMonitor.ComputeCpuStatus()
	memory := h.MemoryMonitor.ComputeMemoryUsage()

	response := Report{
		Cache:        cache,
		Array:        array,
		Parity:       parity,
		Network:      network,
		ArrayTotal:   arrayTotal,
		CacheTotal:   cacheTotal,
		NetworkTotal: networkTotal,
		Cpu:          cpu,
		Memory:       memory,
		Error:        nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error trying to respond to API call",
			slog.String("error", err.Error()),
			slog.String("attempting to marshal", fmt.Sprintf("%+v\n", response)))
		errorResponse, _ := json.Marshal(newErrorReport(err.Error()))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(errorResponse))
	} else {
		slog.Debug("Responding to request", "response", responseJson)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseJson))
	}
}

type Report struct {
	Array        []monitor.DiskStatus   `json:"array"`
	Cache        []monitor.DiskStatus   `json:"cache"`
	Parity       []monitor.ParityStatus `json:"parity"`
	Network      []monitor.NetworkRate  `json:"network"`
	ArrayTotal   monitor.DiskStatus     `json:"array_total"`
	CacheTotal   monitor.DiskStatus     `json:"cache_total"`
	NetworkTotal monitor.NetworkRate    `json:"network_total"`
	Cpu          monitor.CpuStatus      `json:"cpu"`
	Memory       monitor.MemoryStatus   `json:"memory"`
	Error        *string                `json:"error"`
}

func newErrorReport(err string) (report Report) {

	report.Array = make([]monitor.DiskStatus, 0)
	report.Cache = make([]monitor.DiskStatus, 0)
	report.Network = make([]monitor.NetworkRate, 0)

	report.Error = &err

	return
}
