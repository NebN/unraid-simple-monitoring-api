package monitor

import (
	"os"
	"path/filepath"
	"sync"

	"log/slog"

	"github.com/NebN/unraid-simple-monitoring-api/internal/util"
	"github.com/shirou/gopsutil/disk"
)

type DiskStatus struct {
	Path        string  `json:"mount"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	FreePercent float64 `json:"free_percent"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskMonitor struct {
	cache []string
	array []string
}

func NewDiskMonitor(cache []string, array []string) (dm DiskMonitor) {
	dm.cache = cache
	dm.array = array
	return
}

func (monitor *DiskMonitor) ComputeDiskUsage() ([]DiskStatus, []DiskStatus) {

	var wg sync.WaitGroup

	computeGroup := func(paths []string) []DiskStatus {
		diskChan := make(chan util.IndexedValue[DiskStatus], len(paths))

		for i, path := range paths {
			wg.Add(1)
			go diskUsage(i, path, &wg, diskChan)
		}

		wg.Wait()
		close(diskChan)

		disks := make([]DiskStatus, len(paths))
		for disk := range diskChan {
			disks[disk.Index] = disk.Value
		}

		return disks
	}

	cache := computeGroup(monitor.cache)
	array := computeGroup(monitor.array)

	return cache, array
}

func diskUsage(index int, path string, wg *sync.WaitGroup, diskChan chan util.IndexedValue[DiskStatus]) {
	defer wg.Done()

	var pathToQuery = path
	var hostFsPrefix, isSet = os.LookupEnv("HOSTFS_PREFIX")
	if isSet {
		slog.Debug("Host prefix is set to", "value", hostFsPrefix)
		pathToQuery = filepath.Join(hostFsPrefix, path)
	}
	usage, err := disk.Usage(pathToQuery)

	if err != nil {
		slog.Error("Cannot read disk", slog.String("path", path), slog.String("error", err.Error()))
		diskChan <- util.IndexedValue[DiskStatus]{Index: index, Value: DiskStatus{Path: path}}
	} else {
		total := util.BytesToGibiBytes(usage.Total)
		free := util.BytesToGibiBytes(usage.Free)
		used := total - free
		freePercent := util.RoundTwoDecimals((float64(free) / float64(total)) * 100)
		usedPercent := util.RoundTwoDecimals(100 - freePercent)

		status := DiskStatus{
			Path:        path,
			Total:       total,
			Free:        free,
			Used:        used,
			FreePercent: freePercent,
			UsedPercent: usedPercent,
		}

		diskChan <- util.IndexedValue[DiskStatus]{Index: index, Value: status}
	}
}

func AggregateDiskStatuses(disks []DiskStatus) (status DiskStatus) {
	for _, disk := range disks {
		status.Total = status.Total + disk.Total
		status.Used = status.Used + disk.Used
	}
	status.Free = status.Total - status.Used
	if status.Total != 0 {
		status.FreePercent = util.RoundTwoDecimals(float64(status.Free) / float64(status.Total) * 100)
		status.UsedPercent = util.RoundTwoDecimals(100 - status.FreePercent)
	}
	status.Path = "total"
	return
}
