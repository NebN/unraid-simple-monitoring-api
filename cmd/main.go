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
	Networks     []string            `yaml:"networks"`
	Disks        map[string][]string `yaml:"disks"`
	LoggingLevel string              `yaml:"loggingLevel"`
	CpuTemp      *string             `yaml:"cpuTemp"`
	Include      []string            `yaml:"include"`
	Exclude      []string            `yaml:"exclude"`
	Cors         *Cors               `yaml:"cors"`
}

type Cors struct {
	Origin  string `yaml:"origin"`
	Methods string `yaml:"methods"`
	Headers string `yaml:"headers"`
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
	Cors           *Cors
}

func NewHandler(conf Conf) (handler handler) {
	handler.DiskMonitor = monitor.NewDiskMonitor(conf.Disks)
	handler.NetworkMonitor = monitor.NewNetworkMonitor(conf.Networks)
	handler.CpuMonitor = monitor.NewCpuMonitor(conf.CpuTemp)
	handler.Cors = conf.Cors
	return
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	slog.Debug("Request received", slog.String("request", fmt.Sprintf("%+v\n", r)))

	diskUsage := h.DiskMonitor.ComputeDiskUsage()
	network := h.NetworkMonitor.ComputeNetworkRate()
	cacheTotal := monitor.AggregateDiskStatuses(diskUsage.Cache)
	arrayTotal := monitor.AggregateDiskStatuses(diskUsage.Array)
	networkTotal := monitor.AggregateNetworkRates(network)
	cpu, cores := h.CpuMonitor.ComputeCpuStatus()
	memory := h.MemoryMonitor.ComputeMemoryUsage()

	response := Report{
		Cache:        diskUsage.Cache,
		Array:        diskUsage.Array,
		Pools:        diskUsage.Pools,
		Parity:       diskUsage.Party,
		Network:      network,
		ArrayTotal:   arrayTotal,
		CacheTotal:   cacheTotal,
		NetworkTotal: networkTotal,
		Cpu:          cpu,
		Cores:        cores,
		Memory:       memory,
		Error:        nil,
	}

	w.Header().Set("Content-Type", "application/json")
	if h.Cors != nil {
		w.Header().Set("Access-Control-Allow-Origin", h.Cors.Origin)
		w.Header().Set("Access-Control-Allow-Methods", h.Cors.Methods)
		w.Header().Set("Access-Control-Allow-Headers", h.Cors.Headers)
	}
	responseJson, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error trying to respond to API call",
			slog.String("error", err.Error()),
			slog.String("attempting to marshal", fmt.Sprintf("%+v\n", response)))
		errorResponse, _ := json.Marshal(newErrorReport(err.Error()))
		w.Write([]byte(errorResponse))
	} else {
		slog.Debug("Responding to request", "response", responseJson)
		w.Write([]byte(responseJson))
	}
}

type Report struct {
	Array        []monitor.DiskStatus   `json:"array"`
	Cache        []monitor.DiskStatus   `json:"cache"`
	Pools        []monitor.PoolStatus   `json:"pools"`
	Parity       []monitor.ParityStatus `json:"parity"`
	Network      []monitor.NetworkRate  `json:"network"`
	ArrayTotal   monitor.DiskStatus     `json:"array_total"`
	CacheTotal   monitor.DiskStatus     `json:"cache_total"`
	NetworkTotal monitor.NetworkRate    `json:"network_total"`
	Cpu          monitor.CpuStatus      `json:"cpu"`
	Cores        []monitor.CoreStatus   `json:"cores"`
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
