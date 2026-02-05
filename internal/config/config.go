// Package config defines the server configuration and common utilities.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Temperature unit constants.
const (
	UnitCelsius    = "celsius"
	UnitFahrenheit = "fahrenheit"
	UnitKelvin     = "kelvin"
)

// Config holds the server configuration from CLI args
type Config struct {
	TempUnit       string
	MaxProcesses   int
	MountPoints    []string
	Interfaces     []string
	EnableGPU      bool
	MountPointsStr string
	InterfacesStr  string
}

// Validate checks the configuration and parses string lists
func (c *Config) Validate() error {
	// Validate temperature unit
	c.TempUnit = strings.ToLower(c.TempUnit)
	if c.TempUnit != UnitCelsius && c.TempUnit != UnitFahrenheit && c.TempUnit != UnitKelvin {
		return fmt.Errorf("invalid temp-unit: %s (must be celsius, fahrenheit, or kelvin)", c.TempUnit)
	}

	// Validate max processes
	if c.MaxProcesses < 1 {
		c.MaxProcesses = 10
	}
	if c.MaxProcesses > 50 {
		c.MaxProcesses = 50
	}

	// Parse mount points
	if c.MountPointsStr != "" {
		c.MountPoints = SplitAndTrim(c.MountPointsStr)
	}

	// Parse interfaces
	if c.InterfacesStr != "" {
		c.Interfaces = SplitAndTrim(c.InterfacesStr)
	}

	return nil
}

// SplitAndTrim splits a comma-separated string and trims whitespace
func SplitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ConvertTemperature converts Celsius to the requested unit
func ConvertTemperature(celsius float64, unit string) map[string]float64 {
	result := map[string]float64{
		UnitCelsius: celsius,
	}

	switch strings.ToLower(unit) {
	case UnitFahrenheit:
		result[UnitFahrenheit] = celsius*9/5 + 32
	case UnitKelvin:
		result[UnitKelvin] = celsius + 273.15
	default:
		// Already in celsius
	}

	return result
}

// GetRaspberryPiTemp reads CPU temperature from Pi thermal zone
func GetRaspberryPiTemp() (float64, bool) {
	// Try different thermal zone paths
	paths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
	}

	for _, path := range paths {
		// Clean the path to satisfy gosec G304 (though we trust these hardcoded paths)
		path = filepath.Clean(path)
		data, err := os.ReadFile(path)
		if err == nil {
			tempStr := strings.TrimSpace(string(data))
			tempMilli, err := strconv.ParseFloat(tempStr, 64)
			if err == nil {
				return tempMilli / 1000.0, true
			}
		}
	}

	return 0, false
}

// GetRaspberryPiGPUTemp reads GPU temperature using vcgencmd
func GetRaspberryPiGPUTemp() (float64, bool) {
	cmd := exec.Command("vcgencmd", "measure_temp")
	output, err := cmd.Output()
	if err != nil {
		return 0, false
	}

	// Output format: temp=45.2'C
	outputStr := string(output)
	if strings.HasPrefix(outputStr, "temp=") {
		// Extract number between "temp=" and "'C"
		start := 5
		end := strings.Index(outputStr, "'C")
		if end > start {
			tempStr := outputStr[start:end]
			temp, err := strconv.ParseFloat(tempStr, 64)
			if err == nil {
				return temp, true
			}
		}
	}

	return 0, false
}

// GetThrottledStatus reads Pi throttling status
func GetThrottledStatus() (map[string]interface{}, bool) {
	cmd := exec.Command("vcgencmd", "get_throttled")
	output, err := cmd.Output()
	if err != nil {
		return nil, false
	}

	// Parse throttled value
	outputStr := strings.TrimSpace(string(output))
	if !strings.HasPrefix(outputStr, "throttled=0x") {
		return nil, false
	}

	hexStr := strings.TrimPrefix(outputStr, "throttled=0x")
	value, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return nil, false
	}

	// Decode throttling flags
	return map[string]interface{}{
		"under_voltage_now":      value&0x1 != 0,
		"arm_frequency_capped":   value&0x2 != 0,
		"currently_throttled":    value&0x4 != 0,
		"soft_temp_limit_active": value&0x8 != 0,
		"under_voltage_occurred": value&0x10000 != 0,
		"freq_capped_occurred":   value&0x20000 != 0,
		"throttling_occurred":    value&0x40000 != 0,
		"soft_temp_occurred":     value&0x80000 != 0,
		"raw_value":              hexStr,
	}, true
}

// BytesToHuman converts bytes to human-readable format
func BytesToHuman(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
