package status

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"slatessh/backend/internal/models"
)

type Monitor struct {
	mu       sync.Mutex
	previous map[string]netSnapshot
}

type netSnapshot struct {
	rx        float64
	tx        float64
	timestamp time.Time
}

// NewMonitor 用于执行 NewMonitor 相关后端逻辑。
// 输入参数：无。
// 输出参数：返回 *Monitor。
func NewMonitor() *Monitor {
	return &Monitor{previous: make(map[string]netSnapshot)}
}

// Fetch 用于通过 SSH 采集远端服务器状态。
// 输入参数：sessionID 表示sessionID 参数；client 表示SSH 客户端。
// 输出参数：返回 models.ServerStatus, error；error 表示执行失败原因。
func (m *Monitor) Fetch(sessionID string, client *ssh.Client) (models.ServerStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status := models.ServerStatus{Timestamp: time.Now()}
	if output, err := run(client, "cat /etc/os-release 2>/dev/null"); err == nil {
		status.OSName = parseOSRelease(output)
	}
	if output, err := run(client, "(ip -o -4 addr show scope global 2>/dev/null | awk '{print $2, $4}') || (hostname -I 2>/dev/null | tr ' ' '\\n')"); err == nil {
		status.ServerIP = parseServerIP(output)
	}
	if output, err := run(client, "cat /proc/cpuinfo | grep 'model name' | head -n 1"); err == nil {
		status.CPUModel = parseCPUModel(output)
	}
	if output, err := run(client, "free -m || free"); err == nil {
		parseMemory(output, &status)
	}
	if output, err := run(client, "df -kP /"); err == nil {
		parseDisk(output, &status)
	}
	if output, err := run(client, "cat /proc/stat | head -n 1"); err == nil {
		parseCPUPercent(output, &status)
	}
	if output, err := run(client, "cat /proc/net/dev"); err == nil {
		parseNetwork(sessionID, output, status.Timestamp, &status, m.previous)
	}
	return status, nil
}

// run 用于在远端主机执行状态采集命令。
// 输入参数：client 表示SSH 客户端；command 表示command 参数。
// 输出参数：返回 string, error；error 表示执行失败原因。
func run(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(command); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf(strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

// parseOSRelease 用于解析系统发行版名称。
// 输入参数：output 表示output 参数。
// 输出参数：返回 string。
func parseOSRelease(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return "Unknown"
}

// parseServerIP 用于解析服务器 IPv4 地址。
// 输入参数：output 表示output 参数。
// 输出参数：返回 string。
func parseServerIP(output string) string {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		candidate := fields[len(fields)-1]
		candidate = strings.Split(candidate, "/")[0]
		if ipv4.MatchString(candidate) && !strings.HasPrefix(candidate, "127.") {
			return candidate
		}
	}
	return ""
}

var ipv4 = regexp.MustCompile(`^\d{1,3}(?:\.\d{1,3}){3}$`)

// parseCPUModel 用于解析 CPU 型号。
// 输入参数：output 表示output 参数。
// 输出参数：返回 string。
func parseCPUModel(output string) string {
	parts := strings.SplitN(strings.TrimSpace(output), ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(output)
}

// parseMemory 用于解析内存使用率。
// 输入参数：output 表示output 参数；status 表示HTTP 状态码。
// 输出参数：无。
func parseMemory(output string, status *models.ServerStatus) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		switch fields[0] {
		case "Mem:":
			total, _ := strconv.ParseFloat(fields[1], 64)
			used, _ := strconv.ParseFloat(fields[2], 64)
			status.MemTotalMB = total
			status.MemUsedMB = used
			if total > 0 {
				status.MemPercent = used / total * 100
			}
		case "Swap:":
			total, _ := strconv.ParseFloat(fields[1], 64)
			used, _ := strconv.ParseFloat(fields[2], 64)
			status.SwapTotalMB = total
			status.SwapUsedMB = used
			if total > 0 {
				status.SwapPercent = used / total * 100
			}
		}
	}
}

// parseDisk 用于解析磁盘使用率。
// 输入参数：output 表示output 参数；status 表示HTTP 状态码。
// 输出参数：无。
func parseDisk(output string, status *models.ServerStatus) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 6 {
		return
	}
	total, _ := strconv.ParseFloat(fields[1], 64)
	used, _ := strconv.ParseFloat(fields[2], 64)
	status.DiskTotalKB = total
	status.DiskUsedKB = used
	if total > 0 {
		status.DiskPercent = used / total * 100
	}
}

// parseCPUPercent 用于解析 CPU 使用率快照。
// 输入参数：output 表示output 参数；status 表示HTTP 状态码。
// 输出参数：无。
func parseCPUPercent(output string, status *models.ServerStatus) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 5 {
		return
	}
	nums := make([]float64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseFloat(field, 64)
		if err == nil {
			nums = append(nums, value)
		}
	}
	if len(nums) == 0 {
		return
	}
	total := 0.0
	for _, n := range nums {
		total += n
	}
	idle := nums[3]
	if total > 0 {
		status.CPUPercent = (1 - idle/total) * 100
	}
}

// parseNetwork 用于解析网络收发速率。
// 输入参数：sessionID 表示sessionID 参数；output 表示output 参数；now 表示now 参数；status 表示HTTP 状态码；previous 表示previous 参数。
// 输出参数：无。
func parseNetwork(sessionID, output string, now time.Time, status *models.ServerStatus, previous map[string]netSnapshot) {
	lines := strings.Split(output, "\n")
	bestRX := 0.0
	bestTX := 0.0
	for _, line := range lines[2:] {
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if strings.HasPrefix(iface, "lo") {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		rx, _ := strconv.ParseFloat(fields[0], 64)
		tx, _ := strconv.ParseFloat(fields[8], 64)
		if rx+tx > bestRX+bestTX {
			bestRX, bestTX = rx, tx
		}
	}
	prev, ok := previous[sessionID]
	previous[sessionID] = netSnapshot{rx: bestRX, tx: bestTX, timestamp: now}
	if !ok {
		return
	}
	duration := now.Sub(prev.timestamp).Seconds()
	if duration <= 0 {
		return
	}
	status.NetRxRate = (bestRX - prev.rx) / duration
	status.NetTxRate = (bestTX - prev.tx) / duration
}
