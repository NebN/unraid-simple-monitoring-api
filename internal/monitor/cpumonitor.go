package monitor

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/NebN/unraid-simple-monitoring-api/internal/util"
)

type CpuStatus struct {
	LoadPercent float64 `json:"load_percent"`
}

type CpuSnapshot struct {
	idle  uint64
	total uint64
}

type CpuMonitor struct {
	snapshot CpuSnapshot
	mu       sync.Mutex
}

func NewCpuMonitor() (cm CpuMonitor) {
	cm.snapshot = newCpuSnapshot()
	return
}

func (m *CpuMonitor) ComputeCpuStatus() (status CpuStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot := newCpuSnapshot()
	oldSnapshot := m.snapshot

	deltaIdle := snapshot.idle - oldSnapshot.idle
	deltaTotal := snapshot.total - oldSnapshot.total

	slog.Debug("CPU snapshot delta", "idle", deltaIdle, "total", deltaTotal)
	loadPercent := 0.0
	if deltaTotal > 0 {
		loadPercent = (1 - (float64(deltaIdle) / float64(deltaTotal))) * 100
	} else {
		slog.Warn("CPU delta between snapshots' total values is 0, cpu load percent will be returned as 0")
	}

	status.LoadPercent = util.RoundTwoDecimals(loadPercent)
	m.snapshot = snapshot

	slog.Debug("CPU status computed", "status", status)
	return
}

func newCpuSnapshot() (snapshot CpuSnapshot) {
	stat, err := os.Open("/proc/stat")
	if err != nil {
		slog.Error("CPU Cannot read data", slog.String("error", err.Error()))
	}
	defer stat.Close()

	scanner := bufio.NewScanner(stat)
	if !scanner.Scan() {
		slog.Error("CPU unable to read /proc/stat")
		return
	}

	firstLine := scanner.Text()
	slog.Debug("CPU", "line", firstLine)
	items := strings.Fields(firstLine)
	var sum uint64 = 0
	for i, item := range items[1:] {
		parsed, err := strconv.ParseUint(item, 10, 64)
		if err != nil {
			slog.Error("CPU cannot parse cpu data from /proc/stat",
				slog.String("trying to parse", item),
				slog.String("error", err.Error()))
		}
		sum += parsed
		slog.Debug("CPU parsed", "value", parsed)
		if i == 3 {
			slog.Debug("CPU idle value found", "idle", parsed)
			snapshot.idle = parsed
		}
	}

	snapshot.total = sum

	slog.Debug("CPU", "snapshot", snapshot)
	return
}
