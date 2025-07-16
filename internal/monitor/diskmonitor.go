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

	"github.com/NebN/unraid-simple-monitoring-api/internal/conf"
	"github.com/NebN/unraid-simple-monitoring-api/internal/util"
	"github.com/shirou/gopsutil/disk"
	"gopkg.in/ini.v1"
)

const parityLabel = "parity"
const arrayLabel = "array"
const cacheLabel = "cache"

type Pool struct {
	Name   string   `yaml:"name"`
	Mounts []string `yaml:"mounts"`
}

type DiskStatus struct {
	Name        string  `json:"-"`
	Path        string  `json:"mount"`
	Total       float64 `json:"total"`
	Used        float64 `json:"used"`
	Free        float64 `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	FreePercent float64 `json:"free_percent"`
	Temp        uint64  `json:"temp"`
	Id          string  `json:"disk_id"`
	IsSpinning  bool    `json:"is_spinning"`
}

type ParityStatus struct {
	Name       string `json:"name"`
	Temp       uint64 `json:"temp"`
	Id         string `json:"disk_id"`
	IsSpinning bool   `json:"is_spinning"`
}

type PoolStatus struct {
	Name  string       `json:"name"`
	Total DiskStatus   `json:"total"`
	Disks []DiskStatus `json:"disks"`
}

type DiskIni struct {
	Id       string
	Temp     uint64
	Spundown bool
}

type DiskMonitor struct {
	pools              []Pool
	checkZfs           bool
	bytesToCorrectUnit util.MapWithDefault[string, func(float64) float64]
}

type ZfsDataset struct {
	Name       string
	Used       string
	Avail      string
	Refer      string
	Mountpoint string
}

func NewDiskMonitor(disks map[string][]string, units conf.Units) (dm DiskMonitor) {
	var conversionFunctionsMap = make(map[string]func(float64) float64)

	conversionFunctionsMap[arrayLabel] = util.SizeConvertionFunction(util.BYTE, units.Array)

	conversionFunctionsMap[cacheLabel] = util.SizeConvertionFunction(util.BYTE, units.Cache)

	conversionFunctionsMapWithDefault := util.NewMapWithDefault(
		conversionFunctionsMap,
		util.SizeConvertionFunction(util.BYTE, units.Pools),
	)

	dm.bytesToCorrectUnit = conversionFunctionsMapWithDefault

	pools := make([]Pool, 0, len(disks))
	for name, mounts := range disks {
		pools = append(pools, Pool{
			Name:   name,
			Mounts: mounts,
		})
	}
	dm.pools = pools

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

type DiskUsage struct {
	Array []DiskStatus
	Cache []DiskStatus
	Party []ParityStatus
	Pools []PoolStatus
}

func (monitor *DiskMonitor) ComputeDiskUsage() DiskUsage {
	diskIniMap := readDiskIni()

	var wg sync.WaitGroup

	zfsDatasets := monitor.readZfsDatasets()

	computeGroup := func(pool Pool) []DiskStatus {
		diskChan := make(chan util.IndexedValue[DiskStatus], len(pool.Mounts))

		for i, path := range pool.Mounts {
			dataset, exists := zfsDatasets[path]
			if exists {
				diskChan <- util.IndexedValue[DiskStatus]{
					Index: i,
					Value: zfsDatasetUsage(dataset, monitor.bytesToCorrectUnit.Get(pool.Name)),
				}
			} else {
				wg.Add(1)
				go diskUsage(i, path, monitor.bytesToCorrectUnit.Get(pool.Name), &wg, diskChan)
			}
		}

		wg.Wait()
		close(diskChan)

		disks := make([]DiskStatus, len(pool.Mounts))
		for disk := range diskChan {
			diskIni := diskIniMap[disk.Value.Name]

			disk.Value.Id = diskIni.Id
			disk.Value.Temp = diskIni.Temp
			disk.Value.IsSpinning = !diskIni.Spundown

			disks[disk.Index] = disk.Value
		}

		return disks
	}

	poolsStatus := make([]PoolStatus, 0, len(monitor.pools))
	var array PoolStatus
	var cache PoolStatus

	for _, pool := range monitor.pools {
		disks := computeGroup(pool)
		total := AggregateDiskStatuses(disks)
		status := PoolStatus{
			Name:  pool.Name,
			Total: total,
			Disks: disks,
		}
		if pool.Name == arrayLabel {
			array = status
		} else if pool.Name == cacheLabel {
			cache = status
		} else {
			poolsStatus = append(poolsStatus, status)
		}
	}

	parity := make([]ParityStatus, 0)
	for name, diskIni := range diskIniMap {
		if strings.Contains(name, parityLabel) {
			parity = append(parity, ParityStatus{
				Name:       name,
				Temp:       diskIni.Temp,
				Id:         diskIni.Id,
				IsSpinning: !diskIni.Spundown,
			})
		}
	}

	sort.Slice(parity, func(i, j int) bool {
		return parity[i].Name < parity[j].Name
	})
	return DiskUsage{
		Array: array.Disks,
		Cache: cache.Disks,
		Pools: poolsStatus,
		Party: parity,
	}
}

func diskUsage(
	index int,
	path string,
	bytesToUnit func(float64) float64,
	wg *sync.WaitGroup,
	diskChan chan util.IndexedValue[DiskStatus]) {

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
		total := bytesToUnit(float64(usage.Total))
		free := bytesToUnit(float64(usage.Free))
		used := total - free

		freePercent := 0.0
		usedPercent := 0.0

		if total > 0 {
			freePercent = (free / total) * 100
			usedPercent = 100 - freePercent
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

func zfsDatasetUsage(dataset ZfsDataset, bytesToUnit func(float64) float64) DiskStatus {
	usedBytes, _ := util.ParseZfsSize(dataset.Used)
	freeBytes, _ := util.ParseZfsSize(dataset.Avail)
	used := bytesToUnit(usedBytes)
	free := bytesToUnit(freeBytes)
	total := usedBytes + freeBytes

	freePercent := 0.0
	usedPercent := 0.0

	if total > 0 {
		freePercent = (free / total) * 100
		usedPercent = 100 - freePercent
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
		return make(map[string]DiskIni)
	}

	for _, section := range disks.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}

		sectionName := strings.Trim(section.Name(), "\"")

		slog.Debug("Disk reading section", "section", sectionName)

		idKey := "id"
		idString, err := section.GetKey(idKey)
		if err != nil {
			slog.Error("Disk unable to read disks.ini section", "section", sectionName, "key", idKey, "error", slog.String("error", err.Error()))
			continue
		}

		var temp uint64
		tempKey := "temp"
		tempString, err := section.GetKey(tempKey)
		if err != nil {
			slog.Error("Disk unable to read disks.ini section", "section", sectionName, "key", tempKey, "error", slog.String("error", err.Error()))
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

		spunDownKey := "spundown"
		spunDown, err := section.GetKey(spunDownKey)
		if err != nil {
			slog.Error("Disk unable to read disks.ini section", "section", sectionName, "key", spunDownKey, "error", slog.String("error", err.Error()))
			continue
		}

		diskIniMap[sectionName] = DiskIni{
			Id:       idString.String(),
			Temp:     temp,
			Spundown: spunDown.String() == "1",
		}
	}

	return diskIniMap
}

func AggregateDiskStatuses(disks []DiskStatus) (status DiskStatus) {
	paths := make([]string, 0, len(disks))
	temps := make([]float64, 0)
	ids := make([]string, 0, len(disks))
	isSpinning := false

	for _, disk := range disks {
		paths = append(paths, disk.Path)
		ids = append(ids, disk.Id)
		isSpinning = isSpinning || disk.IsSpinning
		if disk.Temp > 0 {
			temps = append(temps, float64(disk.Temp))
		}
		status.Total = status.Total + disk.Total
		status.Used = status.Used + disk.Used
		slog.Debug("Disk aggregating usage", "current_disk", disk, "running_total", status)
	}

	status.Free = status.Total - status.Used

	if status.Total > 0 {
		status.FreePercent = (status.Free / status.Total) * 100
		status.UsedPercent = 100 - status.FreePercent
	} else {
		slog.Warn("Disk aggregation total space is 0, free/used percent will be returned as 0")
	}

	slog.Debug("Disk figuring out common base path", "paths", paths)
	status.Path = util.CommonBase(paths...) + "*"
	slog.Debug("Disk common base path", "path", status.Path)

	status.Temp = uint64(util.Average(temps))
	status.IsSpinning = isSpinning
	status.Id = strings.Join(ids, " ")

	return
}
