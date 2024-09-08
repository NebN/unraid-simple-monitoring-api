package monitor

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"log/slog"

	"github.com/NebN/unraid-simple-monitoring-api/internal/util"
	"github.com/shirou/gopsutil/disk"
	"gopkg.in/ini.v1"
)

const parityLabel = "parity"

type DiskStatus struct {
	Name        string  `json:"-"`
	Path        string  `json:"mount"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	FreePercent float64 `json:"free_percent"`
	Temp        uint64  `json:"temp"`
	Id          string  `json:"disk_id"`
	IsSpinning  bool    `json:"is_spinning"`
}

type ParityStatus struct {
	Name 		string `json:"name"`
	Temp 		uint64 `json:"temp"`
	Id   		string  `json:"disk_id"`
	IsSpinning  bool    `json:"is_spinning"`
}

type DiskIni struct {
    ID   string
    Temp uint64
	Spundown bool
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
		slog.Error("Disk unable to parse env variable as bool",
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

func (monitor *DiskMonitor) ComputeDiskUsage() ([]DiskStatus, []DiskStatus, []ParityStatus) {
	diskIniMap := readDiskIni()

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
			diskIni := diskIniMap[disk.Value.Name]
			
			disk.Value.Id = diskIni.ID
			disk.Value.Temp = diskIni.Temp
			disk.Value.IsSpinning = !diskIni.Spundown

			disks[disk.Index] = disk.Value
		}

		return disks
	}

	cache := computeGroup(monitor.cache)
	array := computeGroup(monitor.array)

	parity := make([]ParityStatus, 0)
	for name, diskIni := range diskIniMap {
		if strings.Contains(name, parityLabel) {
			parity = append(parity, ParityStatus{
				Name: name,
				Temp: diskIni.Temp,
				Id: diskIni.ID,
				IsSpinning: !diskIni.Spundown,
			})
		}
	}

	sort.Slice(parity, func(i, j int) bool {
		return parity[i].Name < parity[j].Name
	})
	return cache, array, parity
}

func diskUsage(index int, path string, wg *sync.WaitGroup, diskChan chan util.IndexedValue[DiskStatus]) {
	defer wg.Done()

	var pathToQuery = path

	var hostFsPrefix, isSet = os.LookupEnv("HOSTFS_PREFIX")
	if isSet {
		slog.Debug("Disk host prefix is set", "value", hostFsPrefix)
		pathToQuery = filepath.Join(hostFsPrefix, path)
	}
	slog.Debug("Disk reading usage", "path", pathToQuery, "original_path", path)
	usage, err := disk.Usage(pathToQuery)

	if err != nil {
		slog.Error("Disk cannot read", slog.String("path", path), slog.String("error", err.Error()))
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
			slog.Warn("Disk total size is 0, free/used percent will be returned as 0", slog.String("disk", path))
		}

		status := DiskStatus{
			Name:        filepath.Base(path),
			Path:        path,
			Total:       total,
			Free:        free,
			Used:        used,
			FreePercent: freePercent,
			UsedPercent: usedPercent,
		}

		slog.Debug("Disk status computed", "index", index, "status", status)

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
		slog.Warn("Disk ZFS dataset total size is 0, free/used percent will be returned as 0", slog.String("dataset", dataset.Mountpoint))
	}

	status := DiskStatus{
		Name:        filepath.Base(dataset.Mountpoint),
		Path:        dataset.Mountpoint,
		Total:       total,
		Free:        free,
		Used:        used,
		FreePercent: freePercent,
		UsedPercent: usedPercent,
	}

	slog.Debug("Disk ZFS dataset status computed", "status", status)

	return status
}

func (monitor *DiskMonitor) readZfsDatasets() map[string]ZfsDataset {

	zfsDatasets := make(map[string]ZfsDataset)

	if !monitor.checkZfs {
		slog.Debug("Disk ZFS dataset checking is disabled")
		return zfsDatasets
	} else {
		slog.Debug("Disk ZFS dataset checking is enabled")
	}

	cmd := exec.Command("zfs", "list")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Disk error while preparing command 'zfs list'", slog.String("error", err.Error()))
		return zfsDatasets
	}

	if err := cmd.Start(); err != nil {
		slog.Error("Disk error running command 'zfs list'", slog.String("error", err.Error()))
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
		slog.Debug("Disk ZFS dataset found", "dataset", ds)
		zfsDatasets[ds.Mountpoint] = ds
	}
	cmd.Wait()

	return zfsDatasets
}

func readDiskIni() map[string]DiskIni {
    pathToQuery := "/var/local/emhttp/disks.ini"

    var hostFsPrefix, isSet = os.LookupEnv("HOSTFS_PREFIX")
    if isSet {
        slog.Debug("Disk host prefix is set", "value", hostFsPrefix)
        pathToQuery = filepath.Join(hostFsPrefix, pathToQuery)
    }

    diskIniMap := make(map[string]DiskIni)

    disks, err := ini.Load(pathToQuery)
    if err != nil {
        slog.Error("Disk unable to read disks.ini", "error", slog.String("error", err.Error()))
    }

    for _, section := range disks.Sections() {
        if section.Name() == "DEFAULT" {
            continue
        }

        sectionName := strings.Trim(section.Name(), "\"")

        slog.Debug("Disk reading section", "section", sectionName)

        idString, err := section.GetKey("id")
        if err != nil {
            slog.Error("Disk unable to read disks.ini section", "section", sectionName, "error", slog.String("error", err.Error()))
            continue
        }

        var temp uint64
        tempString, err := section.GetKey("temp")
        if err != nil {
            slog.Error("Disk unable to read disks.ini section", "section", sectionName, "error", slog.String("error", err.Error()))
            continue
        }

        if tempString.String() != "*" {
            temp, err = strconv.ParseUint(tempString.String(), 10, 16)
            if err != nil {
                slog.Error("Disk unable to parse disk temp", "string", tempString.String(), "error", slog.String("error", err.Error()))
            }
        } else {
            slog.Debug("Disk temp unavailable", "disk", sectionName)
        }

		spunDown, err := section.GetKey("spundown")

        diskIniMap[sectionName] = DiskIni{
            ID:   idString.String(),
            Temp: temp,
			Spundown: spunDown.String() == "1",
        }
    }

    return diskIniMap
}

func AggregateDiskStatuses(disks []DiskStatus) (status DiskStatus) {
	for _, disk := range disks {
		status.Total = status.Total + disk.Total
		status.Used = status.Used + disk.Used
		slog.Debug("Disk aggregating usage", "current_disk", disk, "running_total", status)
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
