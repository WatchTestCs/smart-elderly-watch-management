package runtuning

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCPUQuotaParser(t *testing.T) {
	for _, tc := range []struct {
		input  string
		online int
		want   int
	}{
		{"max 100000", 8, 0}, {"100000 100000", 8, 1},
		{"250000 100000", 8, 3}, {"300000 100000", 8, 3},
		{"10000 100000", 8, 1}, {"200000 100000", 2, 0},
		{"0 100000", 8, 0}, {"123 0", 8, 0}, {"bad 100000", 8, 0},
	} {
		if got := cpuParallelism(tc.input, tc.online); got != tc.want {
			t.Errorf("cpu(%q,%d)=%d want %d", tc.input, tc.online, got, tc.want)
		}
	}
}
func TestMemoryBudget(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int64
	}{
		{"max", 0}, {"", 0}, {"100000", 0}, {"not a number", 0},
		{"268435456", 187904819}, {"4294967296", 3006477107},
	} {
		if got := goMemoryBudget(tc.input); got != tc.want {
			t.Errorf("memory(%q)=%d want %d", tc.input, got, tc.want)
		}
	}
}
func TestApplyOptInPrecedenceAndV2Files(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cpu.max"), []byte("250000 100000"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "memory.max"), []byte("4294967296"), 0600); err != nil {
		t.Fatal(err)
	}
	var cpu int
	var memory int64
	setCPU := func(n int) int { cpu = n; return n }
	setMemory := func(n int64) int64 { memory = n; return n }
	get := func(k string) string { return "" }
	d := apply(root, "/", get, 8, setCPU, setMemory)
	if d.Parallelism != 3 || d.GoMemoryLimit != 3006477107 || cpu != 3 || memory != 3006477107 {
		t.Fatalf("decision %+v cpu=%d memory=%d", d, cpu, memory)
	}
	cpu, memory = 0, 0
	d = apply(root, "/", func(k string) string { return "administrator setting" }, 8, setCPU, setMemory)
	if d != (Decision{}) || cpu != 0 || memory != 0 {
		t.Fatalf("overwrote explicit environment: %+v", d)
	}
}

func TestNestedLimitsAndExplicitOverrides(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "system.slice", "suining.service")
	if err := os.MkdirAll(leaf, 0700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"cpu.max":                                 "max 100000",
		"memory.max":                              "max",
		"system.slice/cpu.max":                    "150000 100000",
		"system.slice/memory.max":                 "536870912",
		"system.slice/suining.service/cpu.max":    "300000 100000",
		"system.slice/suining.service/memory.max": "max",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	group, ok := unifiedGroup("0::/system.slice/suining.service\n")
	if !ok || group != "/system.slice/suining.service" {
		t.Fatal(group, ok)
	}
	cpu, memory := 0, int64(0)
	setCPU := func(n int) int { cpu = n; return n }
	setMemory := func(n int64) int64 { memory = n; return n }
	get := func(string) string { return "" }
	d := apply(root, group, get, 8, setCPU, setMemory)
	if d.Parallelism != 2 || d.GoMemoryLimit != 375809638 || cpu != 2 || memory != 375809638 {
		t.Fatalf("decision=%+v cpu=%d mem=%d", d, cpu, memory)
	}
	if err := os.WriteFile(filepath.Join(leaf, "memory.max"), []byte("134217728"), 0600); err != nil {
		t.Fatal(err)
	}
	cpu, memory = 0, 0
	d = apply(root, group, func(k string) string {
		if k == "GOMAXPROCS" {
			return "4"
		}
		return ""
	}, 8, setCPU, setMemory)
	if d != (Decision{}) || cpu != 0 || memory != 0 {
		t.Fatalf("explicit CPU or small leaf memory limit ignored: %+v", d)
	}
}
