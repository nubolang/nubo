package system

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/nubolang/nubo/native/n"
	"golang.org/x/term"
)

// helper functions (cross-platform with runtime.GOOS branches)

// getMemory returns totalBytes, freeBytes, error
func getMemory() (int64, int64, error) {
	switch runtime.GOOS {
	case "linux":
		// parse /proc/meminfo
		f, err := os.Open("/proc/meminfo")
		if err != nil {
			return 0, 0, err
		}
		defer f.Close()
		var total, available int64
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			key := strings.TrimSuffix(fields[0], ":")
			val, _ := strconv.ParseInt(fields[1], 10, 64)
			switch key {
			case "MemTotal":
				total = val * 1024
			case "MemAvailable":
				available = val * 1024
			}
			if total != 0 && available != 0 {
				break
			}
		}
		if total == 0 {
			return 0, 0, errors.New("could not read MemTotal from /proc/meminfo")
		}
		return total, available, nil

	case "darwin":
		// total: sysctl hw.memsize
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0, 0, err
		}
		totalStr := strings.TrimSpace(string(out))
		total, err := strconv.ParseInt(totalStr, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		// free: parse vm_stat for "Pages free" and "Pages inactive" as a best-effort
		out, err = exec.Command("vm_stat").Output()
		if err != nil {
			// best-effort: return total and 0 free
			return total, 0, nil
		}
		var freePages int64
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Pages free:") || strings.HasPrefix(line, "Pages inactive:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					numStr := strings.TrimSuffix(parts[2], ".")
					n, _ := strconv.ParseInt(numStr, 10, 64)
					freePages += n
				}
			}
		}
		// page size: try sysctl hw.pagesize, fallback to 4096
		pageOut, err := exec.Command("sysctl", "-n", "hw.pagesize").Output()
		pageSize := int64(4096)
		if err == nil {
			if p, err := strconv.ParseInt(strings.TrimSpace(string(pageOut)), 10, 64); err == nil && p > 0 {
				pageSize = p
			}
		}
		return total, freePages * pageSize, nil

	case "windows":
		// Use PowerShell Get-CimInstance -> JSON (TotalVisibleMemorySize, FreePhysicalMemory)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", "Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize,FreePhysicalMemory | ConvertTo-Json -Compress")
		out, err := cmd.Output()
		if err != nil {
			return 0, 0, err
		}
		// JSON looks like: {"TotalVisibleMemorySize":16777216,"FreePhysicalMemory":1234567}
		var m struct {
			TotalVisibleMemorySize int64 `json:"TotalVisibleMemorySize"`
			FreePhysicalMemory     int64 `json:"FreePhysicalMemory"`
		}
		if err := json.Unmarshal(out, &m); err != nil {
			// sometimes ConvertTo-Json returns an array if multiple objects; try array
			var arr []map[string]interface{}
			if err2 := json.Unmarshal(out, &arr); err2 == nil && len(arr) > 0 {
				if v, ok := arr[0]["TotalVisibleMemorySize"].(float64); ok {
					m.TotalVisibleMemorySize = int64(v)
				}
				if v, ok := arr[0]["FreePhysicalMemory"].(float64); ok {
					m.FreePhysicalMemory = int64(v)
				}
			} else {
				return 0, 0, err
			}
		}
		// Values are in KB according to Win32_OperatingSystem docs
		return m.TotalVisibleMemorySize * 1024, m.FreePhysicalMemory * 1024, nil

	default:
		// fallback: use runtime.ReadMemStats as a poor-man's fallback (process memory, not system)
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		return int64(ms.Sys), int64(ms.Frees), nil
	}
}

// getUptime returns seconds since boot
func getUptime() (int64, error) {
	switch runtime.GOOS {
	case "linux":
		// read /proc/uptime -> first field seconds since boot (float)
		data, err := os.ReadFile("/proc/uptime")
		if err != nil {
			return 0, err
		}
		parts := strings.Fields(string(data))
		if len(parts) == 0 {
			return 0, errors.New("invalid /proc/uptime")
		}
		f, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return int64(f), nil

	case "darwin":
		// use sysctl kern.boottime
		out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
		if err != nil {
			// alternative: use sysctl -a and find kern.boottime; but error out for now
			return 0, err
		}
		// output looks like: { sec = 1618888888, usec = 0 } or "sec = 1618888888 usec = 0"
		s := string(out)
		// find first number with 9+ digits (unix timestamp)
		fields := strings.FieldsFunc(s, func(r rune) bool {
			return !(r >= '0' && r <= '9')
		})
		for _, f := range fields {
			if len(f) >= 9 { // a heuristic for seconds
				if sec, err := strconv.ParseInt(f, 10, 64); err == nil {
					return time.Now().Unix() - sec, nil
				}
			}
		}
		return 0, errors.New("unable to parse kern.boottime")

	case "windows":
		// use PowerShell Get-CimInstance Win32_OperatingSystem and LastBootUpTime
		// We'll ask PowerShell to return LastBootUpTime as a string in ISO-ish format
		ps := `((Get-CimInstance Win32_OperatingSystem).LastBootUpTime).ToString("yyyyMMddHHmmss")`
		cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
		out, err := cmd.Output()
		if err != nil {
			// fallback: use Get-CimInstance and parse raw format
			cmd2 := exec.Command("powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_OperatingSystem).LastBootUpTime")
			out2, err2 := cmd2.Output()
			if err2 != nil {
				return 0, err
			}
			out = out2
		}
		tStr := strings.TrimSpace(string(out))
		// tStr expected "yyyyMMddHHmmss" e.g. "20250112083030"
		if len(tStr) >= 14 {
			parsed, err := time.ParseInLocation("20060102150405", tStr[:14], time.Local)
			if err == nil {
				return int64(time.Since(parsed).Seconds()), nil
			}
		}
		// last-resort: try to parse first 14 digits from the string
		digits := ""
		for _, ch := range tStr {
			if ch >= '0' && ch <= '9' {
				digits += string(ch)
				if len(digits) >= 14 {
					break
				}
			}
		}
		if len(digits) >= 14 {
			if parsed, err := time.ParseInLocation("20060102150405", digits[:14], time.Local); err == nil {
				return int64(time.Since(parsed).Seconds()), nil
			}
		}
		return 0, errors.New("unable to parse Windows LastBootUpTime")

	default:
		// fallback: return process uptime (time since program start)
		// we can't know process start without global var; approximate: 0
		return 0, errors.New("uptime not implemented for this OS")
	}
}

// --- exported nubo functions ---

var osName = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return runtime.GOOS, nil
})

var arch = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return runtime.GOARCH, nil
})

var hostname = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	return os.Hostname()
})

var user = n.Function(n.Describe().Returns(n.TString), func(a *n.Args) (any, error) {
	// prefer platform-typical env vars
	if runtime.GOOS == "windows" {
		if v := os.Getenv("USERNAME"); v != "" {
			return v, nil
		}
	}
	if v := os.Getenv("USER"); v != "" {
		return v, nil
	}
	if v := os.Getenv("LOGNAME"); v != "" {
		return v, nil
	}
	// last-resort: empty string
	return "", nil
})

var cpu = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	return int64(runtime.NumCPU()), nil
})

var memoryTotal = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	total, _, err := getMemory()
	if err != nil {
		return int64(0), err
	}
	return total, nil
})

var memoryFree = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	_, free, err := getMemory()
	if err != nil {
		return int64(0), err
	}
	return free, nil
})

var uptime = n.Function(n.Describe().Returns(n.TInt), func(a *n.Args) (any, error) {
	return getUptime()
})

var isTTY = n.Function(n.Describe(n.Arg("fd", n.TInt)).Returns(n.TBool), func(a *n.Args) (any, error) {
	// your convention passes ints as int64; convert
	v := a.Name("fd").Value().(int64)
	return term.IsTerminal(int(v)), nil
})
