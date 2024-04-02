package monitor

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	UsedPercent float64 `json:"used_percent"`
	FreePercent float64 `json:"free_percent"`
}

type DiskMonitor struct {
	cache    []string
	array    []string
	checkZfs bool
}

type ZfsDataset struct {
	Name       string
	Used       string
	Avail      string
	Refer      string
	Mountpoint string
}

func NewDiskMonitor(cache []string, array []string) (dm DiskMonitor) {
	dm.cache = cache
	dm.array = array

	checkZfsString := os.Getenv("ZFS_OK")
	checkZfsBool, err := strconv.ParseBool(checkZfsString)

	if err != nil {
		slog.Error("Unable to parse env variable as bool",
			slog.String("variable name", "ZFS_OK"),
			slog.String("variable value", checkZfsString))
	}

	dm.checkZfs = checkZfsBool

	if checkZfsBool {
		slog.Info("Running in privileged mode. Will be able to check zfs datasets.")
	} else {
		slog.Info("Not running in privileged mode. Will not be able to check zfs datasets.")
	}
	return
}

func (monitor *DiskMonitor) ComputeDiskUsage() ([]DiskStatus, []DiskStatus) {

	var wg sync.WaitGroup

	zfsDatasets := monitor.readZfsDatasets()

	computeGroup := func(paths []string) []DiskStatus {
		diskChan := make(chan util.IndexedValue[DiskStatus], len(paths))

		for i, path := range paths {
			dataset, exists := zfsDatasets[path]
			if exists {
				diskChan <- util.IndexedValue[DiskStatus]{
					Index: i,
					Value: zfsDatasetUsage(dataset),
				}
			} else {
				wg.Add(1)
				go diskUsage(i, path, &wg, diskChan)
			}
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

		freePercent := 0.0
		usedPercent := 0.0

		if total > 0 {
			freePercent = util.RoundTwoDecimals((float64(free) / float64(total)) * 100)
			usedPercent = util.RoundTwoDecimals(100 - freePercent)
		} else {
			slog.Warn("Total disk size is 0, free/used percent will be returned as 0", slog.String("disk", path))
		}

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

func zfsDatasetUsage(dataset ZfsDataset) DiskStatus {
	usedBytes, _ := util.ParseZfsSize(dataset.Used)
	used := util.BytesToGibiBytes(usedBytes)
	freeBytes, _ := util.ParseZfsSize(dataset.Avail)
	free := util.BytesToGibiBytes(freeBytes)
	total := used + free

	freePercent := 0.0
	usedPercent := 0.0

	if total > 0 {
		freePercent = util.RoundTwoDecimals((float64(free) / float64(total)) * 100)
		usedPercent = util.RoundTwoDecimals(100 - freePercent)
	} else {
		slog.Warn("ZFS dataset total size is 0, free/used percent will be returned as 0", slog.String("dataset", dataset.Mountpoint))
	}

	status := DiskStatus{
		Path:        dataset.Mountpoint,
		Total:       total,
		Free:        free,
		Used:        used,
		FreePercent: freePercent,
		UsedPercent: usedPercent,
	}

	return status
}

func (monitor *DiskMonitor) readZfsDatasets() map[string]ZfsDataset {

	zfsDatasets := make(map[string]ZfsDataset)

	if !monitor.checkZfs {
		return zfsDatasets
	}

	cmd := exec.Command("zfs", "list")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Error while preparing command 'zfs list'", slog.String("error", err.Error()))
		return zfsDatasets
	}

	if err := cmd.Start(); err != nil {
		slog.Error("Error running command 'zfs list'", slog.String("error", err.Error()))
		return zfsDatasets
	}

	outScanner := bufio.NewScanner(stdout)

	outScanner.Scan() // skip header
	for outScanner.Scan() {
		fields := strings.Fields(outScanner.Text())
		ds := ZfsDataset{
			Name:       fields[0],
			Used:       fields[1],
			Avail:      fields[2],
			Refer:      fields[3],
			Mountpoint: fields[4],
		}
		slog.Debug("ZFS dataset found", "dataset", ds)
		zfsDatasets[ds.Mountpoint] = ds
	}
	cmd.Wait()

	return zfsDatasets
}

func AggregateDiskStatuses(disks []DiskStatus) (status DiskStatus) {
	for _, disk := range disks {
		status.Total = status.Total + disk.Total
		status.Used = status.Used + disk.Used
	}
	status.Free = status.Total - status.Used
	if status.Total > 0 {
		status.FreePercent = util.RoundTwoDecimals(float64(status.Free) / float64(status.Total) * 100)
		status.UsedPercent = util.RoundTwoDecimals(100 - status.FreePercent)
	} else {
		slog.Warn("Disk aggregation total space is 0, free/used percent will be returned as 0")
	}
	status.Path = "total"
	return
}
