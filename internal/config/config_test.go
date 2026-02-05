package config

import (
	"reflect"
	"testing"
)

func TestBytesToHuman(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536, "1.5 KB"},
	}

	for _, tc := range tests {
		result := BytesToHuman(tc.input)
		if result != tc.expected {
			t.Errorf("BytesToHuman(%d) = %s; want %s", tc.input, result, tc.expected)
		}
	}
}

func TestConvertTemperature(t *testing.T) {
	celsius := 100.0

	// Test Fahrenheit
	res := ConvertTemperature(celsius, "fahrenheit")
	if f, ok := res["fahrenheit"]; !ok || f != 212.0 {
		t.Errorf("ConvertTemperature(100, fahrenheit) fahrenheit = %v; want 212.0", f)
	}

	// Test Kelvin
	res = ConvertTemperature(celsius, "kelvin")
	if k, ok := res["kelvin"]; !ok || k != 373.15 {
		t.Errorf("ConvertTemperature(100, kelvin) kelvin = %v; want 373.15", k)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: Config{
				TempUnit:     "celsius",
				MaxProcesses: 10,
			},
			wantErr: false,
		},
		{
			name: "Invalid temp unit",
			config: Config{
				TempUnit: "rankine",
			},
			wantErr: true,
		},
		{
			name: "Max processes correction (low)",
			config: Config{
				TempUnit:     "celsius",
				MaxProcesses: 0,
			},
			wantErr: false, // Should default to 10
		},
		{
			name: "Max processes correction (high)",
			config: Config{
				TempUnit:     "celsius",
				MaxProcesses: 100,
			},
			wantErr: false, // Should cap at 50
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				if tc.name == "Max processes correction (low)" && tc.config.MaxProcesses != 10 {
					t.Errorf("Expected MaxProcesses to be 10, got %d", tc.config.MaxProcesses)
				}
				if tc.name == "Max processes correction (high)" && tc.config.MaxProcesses != 50 {
					t.Errorf("Expected MaxProcesses to be 50, got %d", tc.config.MaxProcesses)
				}
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	input := " a, b , c,,"
	expected := []string{"a", "b", "c"}
	result := SplitAndTrim(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("SplitAndTrim(%q) = %v; want %v", input, result, expected)
	}
}
