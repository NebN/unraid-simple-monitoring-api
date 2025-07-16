package monitor

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CoreStatus struct {
	Name        string  `json:"name"`
	LoadPercent float64 `json:"load_percent"`
}

type CpuStatus struct {
	LoadPercent float64 `json:"load_percent"`
	Temp        int     `json:"temp"`
}

type CpuSnapshot struct {
	name  string
	idle  uint64
	total uint64
}

type CpuMonitor struct {
	snapshot       CpuSnapshot
	coresSnapshots []CpuSnapshot
	mu             sync.Mutex
	cpuTempPath    *string
}

func NewCpuMonitor(cpuTempPath *string) (cm CpuMonitor) {
	cm.cpuTempPath = cpuTempPath
	cm.snapshot, cm.coresSnapshots = newCpuSnapshot()
	cm.cpuTempPath = locateCpuTempFile(cpuTempPath)
	return
}

func (m *CpuMonitor) ComputeCpuStatus() (status CpuStatus, cores []CoreStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, coresSnapshots := newCpuSnapshot()
	oldSnapshot := m.snapshot
	oldCoreSnapshots := m.coresSnapshots

	status.LoadPercent = computeLoad(oldSnapshot, snapshot)
	status.Temp = m.temp()

	for i, coreSnapshot := range coresSnapshots {
		coreStatus := CoreStatus{
			Name:        coreSnapshot.name,
			LoadPercent: computeLoad(oldCoreSnapshots[i], coreSnapshot),
		}
		cores = append(cores, coreStatus)
	}

	m.snapshot = snapshot
	m.coresSnapshots = coresSnapshots

	slog.Debug("CPU status computed", "status", status)
	return
}

func computeLoad(a CpuSnapshot, b CpuSnapshot) float64 {
	deltaIdle := b.idle - a.idle
	deltaTotal := b.total - a.total

	slog.Debug("CPU snapshot delta", "idle", deltaIdle, "total", deltaTotal)
	loadPercent := 0.0
	if deltaTotal > 0 {
		loadPercent = (1 - (float64(deltaIdle) / float64(deltaTotal))) * 100
	} else {
		slog.Warn("CPU delta between snapshots' total values is 0, cpu load percent will be returned as 0")
	}

	return loadPercent
}

func newCpuSnapshot() (cpu CpuSnapshot, cores []CpuSnapshot) {
	stat, err := os.Open("/proc/stat")
	if err != nil {
		slog.Error("CPU Cannot read data", slog.String("error", err.Error()))
	}
	defer stat.Close()

	scanner := bufio.NewScanner(stat)

	for hasNext := scanner.Scan(); hasNext; hasNext = scanner.Scan() {
		line := scanner.Text()
		slog.Debug("CPU", "line", line)
		fields := strings.Fields(line)
		name := fields[0]
		if !strings.Contains(name, ("cpu")) {
			continue
		}

		if name == "cpu" {
			cpu = parseCpuStatLine(fields)
			cpu.name = name
		} else {
			core := parseCpuStatLine(fields)
			core.name = name
			cores = append(cores, core)
		}

	}

	return
}

func parseCpuStatLine(items []string) (snapshot CpuSnapshot) {

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

func (monitor *CpuMonitor) temp() int {
	if monitor.cpuTempPath == nil {
		return 0
	}

	temp, err := readCpuTemp(*monitor.cpuTempPath)
	if err != nil {
		slog.Error("CPU error while reading temperature.", slog.String("error", err.Error()))
		return 0
	}

	return temp
}

func locateCpuTempFile(cpuTempPath *string) *string {
	if cpuTempPath != nil {
		return cpuTempPath
	}

	slog.Info("CPU temperature file not defined, attempting to locate it. " +
		"It can be specified in the configuration file. \"cpuTemp: /path/to/file\"")

	possiblePatterns := []string{
		// "/sys/class/thermal/thermal_zone*/temp", unsure if checking this makes sense
		"/sys/class/hwmon/hwmon*/temp1_input",
	}

	type cpuFileGuess struct {
		path        string
		initialTemp int
		finalTemp   int
		delta       int
	}

	var guesses = make([]cpuFileGuess, 0)

	for _, pattern := range possiblePatterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			slog.Error("CPU Unable to read files", slog.String("pattern", pattern), slog.String("error", err.Error()))
		}

		for _, file := range files {
			cpuTemp, err := readCpuTemp(file)
			if err != nil {
				slog.Warn(err.Error())
			} else {
				guesses = append(guesses, cpuFileGuess{
					path:        file,
					initialTemp: cpuTemp,
				})
			}
		}
	}

	stressCPU(5 * time.Second)

	for i, guess := range guesses {
		newTemp, err := readCpuTemp(guess.path)
		if err != nil {
			slog.Warn(err.Error())
		} else {
			guessAtIndex := &guesses[i]
			guessAtIndex.finalTemp = newTemp
			guessAtIndex.delta = newTemp - guessAtIndex.initialTemp
		}
	}

	var bestGuess cpuFileGuess = cpuFileGuess{
		delta: 0,
	}
	for _, guess := range guesses {
		if guess.delta > bestGuess.delta {
			bestGuess = guess
		}
	}

	if bestGuess.path != "" {
		slog.Info("Best guess for CPU temperature file", "path", bestGuess.path,
			"initial temp", bestGuess.initialTemp,
			"final temp", bestGuess.finalTemp)
		return &bestGuess.path
	} else {
		slog.Warn("Was unable to find a suitable CPU temperature file")
		if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
			slog.Debug("Guesses:")
			for _, guess := range guesses {
				slog.Debug(fmt.Sprintf("%v", guess))
			}
		}
		return nil
	}
}

func readCpuTemp(path string) (int, error) {
	stat, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer stat.Close()

	scanner := bufio.NewScanner(stat)
	if !scanner.Scan() {
		return 0, fmt.Errorf("unable to read file for CPU temp. path=%s", path)
	}

	firstLine := scanner.Text()
	slog.Debug("CPU", "temp line", firstLine)
	parsed, err := strconv.Atoi(firstLine)
	if err != nil {
		return 0, fmt.Errorf("unable to parse CPU temp data. string=%s, error=%s", firstLine, err.Error())
	}

	return parsed / 1000, nil
}

func stressCPU(duration time.Duration) {
	slog.Info("Running a very quick CPU stress test to attempt to locate the temperature file.",
		"duration", duration)

	stress := func(wg *sync.WaitGroup) {
		defer wg.Done()
		end := time.Now().Add(duration)
		for time.Now().Before(end) {
			for i := 0; i < 100000; i++ {
				_ = i * i
			}
		}
	}

	var wg sync.WaitGroup
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	wg.Add(cpus)

	for range cpus {
		go stress(&wg)
	}

	wg.Wait()
}
