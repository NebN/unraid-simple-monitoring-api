package monitor

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NebN/unraid-simple-monitoring-api/internal/util"
)

type CpuStatus struct {
	LoadPercent float64 `json:"load_percent"`
}

type CpuSnapshot struct {
	idle  uint64
	total uint64
	ts    time.Time
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

	loadPercent := (1 - (float64(deltaIdle) / float64(deltaTotal))) * 100

	status.LoadPercent = util.RoundTwoDecimals(loadPercent)
	m.snapshot = snapshot

	return
}

func newCpuSnapshot() (snapshot CpuSnapshot) {
	stat, err := os.Open("/proc/stat")
	if err != nil {
		slog.Error("Cannot read cpu data", slog.String("error", err.Error()))
	}
	defer stat.Close()

	scanner := bufio.NewScanner(stat)
	if !scanner.Scan() {
		slog.Error("Unable to read /proc/stat")
		return
	}

	snapshot.ts = time.Now()
	firstLine := scanner.Text()
	items := strings.Fields(firstLine)
	var sum uint64 = 0
	for i, item := range items[1:] {
		parsed, err := strconv.ParseUint(item, 10, 64)
		if err != nil {
			slog.Error("Cannot parse cpu data from /proc/stat",
				slog.String("trying to parse", item),
				slog.String("error", err.Error()))
		}
		sum += parsed
		if i == 3 {
			snapshot.idle = parsed
		}
	}

	snapshot.total = sum

	return
}
