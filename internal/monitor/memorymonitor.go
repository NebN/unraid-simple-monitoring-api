package monitor

import (
	"bufio"
	"log/slog"
	"os"
	"regexp"
	"strconv"

	"github.com/NebN/unraid-simple-monitoring-api/internal/util"
)

var memTotalRegex = regexp.MustCompile(`MemTotal:\s+(\d+)`)
var memAvailableRegex = regexp.MustCompile(`MemAvailable:\s+(\d+)`)

type MemoryStatus struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	FreePercent float64 `json:"free_percent"`
}

type MemoryMonitor struct{}

func NewMemoryMonitor() (mm MemoryMonitor) {
	return
}

func (monitor *MemoryMonitor) ComputeMemoryUsage() (status MemoryStatus) {

	meminfo, err := os.Open("/proc/meminfo")
	if err != nil {
		slog.Error("Memory cannot read data", slog.String("error", err.Error()))
		return
	}
	defer meminfo.Close()

	findGroup := func(r *regexp.Regexp, s string) (uint64, bool) {
		res := r.FindStringSubmatch(s)
		if len(res) > 1 {
			parsed, err := strconv.ParseUint(res[1], 10, 64)
			if err != nil {
				slog.Error("Memory cannot parse value from /proc/meminfo",
					slog.String("parsing", res[1]),
					slog.String("error", err.Error()))
				return 0, false
			}
			return parsed, true
		}

		return 0, false
	}

	scanner := bufio.NewScanner(meminfo)
	for scanner.Scan() {
		line := scanner.Text()

		slog.Debug("Memory", "line", line)
		if status.Total == 0 {
			memTotal, found := findGroup(memTotalRegex, line)
			if found {
				status.Total = util.KibiBytesToMebiBytes(memTotal)
				slog.Debug("Memory total parsed", "total_MiB", status.Total)
			}
		}

		if status.Free == 0 {
			memAvailable, found := findGroup(memAvailableRegex, line)
			if found {
				status.Free = util.KibiBytesToMebiBytes(memAvailable)
				slog.Debug("Memory free parsed", "free_MiB", status.Free)
			}
		}

		if status.Total != 0 && status.Free != 0 {
			break
		}
	}

	if status.Total == 0 {
		slog.Error("Memory unable to compute usage")
		return
	}

	status.Used = status.Total - status.Free
	if status.Total > 0 {
		status.FreePercent = util.RoundTwoDecimals((float64(status.Free) / float64(status.Total)) * 100)
		status.UsedPercent = util.RoundTwoDecimals(100 - status.FreePercent)
	} else {
		slog.Warn("Memory total is 0, free/used percent will be returned as 0")
	}

	return
}
