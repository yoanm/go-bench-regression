package benchreg

import (
	"bufio"
	"strings"
	"testing"
)

func Test_parseLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		threshold float64
		want      string
	}{
		// Regression detection - various percentages
		{
			name:      "simple regression above threshold",
			line:      "BenchmarkFunc-8      100  101ns ± 1%  120ns ± 0%  +18.81%",
			threshold: 5,
			want:      "  BenchmarkFunc-8 (18.81% slower)",
		},
		{
			name:      "large regression",
			line:      "BenchmarkLarge-4     50   5000ns ± 2%  10000ns ± 1%  +100.00%",
			threshold: 10,
			want:      "  BenchmarkLarge-4 (100.00% slower)",
		},
		{
			name:      "small regression above threshold",
			line:      "BenchmarkSmall-8     1000  100ns ± 0%  105.5ns ± 0%  +5.50%",
			threshold: 5,
			want:      "  BenchmarkSmall-8 (5.50% slower)",
		},

		// Threshold edge cases
		{
			name:      "delta exactly equals threshold - should not match",
			line:      "BenchmarkExact-8     200  200ns ± 1%  220ns ± 0%  +10.00%",
			threshold: 10,
			want:      "", // Exact match uses > not >=, so 10.00 is NOT > 10.00
		},
		{
			name:      "delta just above threshold",
			line:      "BenchmarkAbove-8     200  200ns ± 1%  220.1ns ± 0%  +10.05%",
			threshold: 10,
			want:      "  BenchmarkAbove-8 (10.05% slower)",
		},
		{
			name:      "delta just below threshold",
			line:      "BenchmarkBelow-8     200  200ns ± 1%  219.9ns ± 0%  +9.95%",
			threshold: 10,
			want:      "", // 9.95 is not > 10
		},

		// Negative deltas (improvements - should be ignored)
		{
			name:      "improvement negative delta",
			line:      "BenchmarkFast-8      100  100ns ± 0%  50ns ± 0%  -50.00%",
			threshold: 5,
			want:      "",
		},
		{
			name:      "improvement just below zero",
			line:      "BenchmarkOpt-4       200  200ns ± 1%  100ns ± 1%  -50.50%",
			threshold: 10,
			want:      "",
		},

		// Invalid or malformed inputs
		{
			name:      "line with no delta",
			line:      "BenchmarkNoDelta-8   100  100ns ± 0%  100ns ± 0%",
			threshold: 5,
			want:      "",
		},
		{
			name:      "line with malformed delta double plus - regex still matches the second +",
			line:      "BenchmarkMalform-8   100  100ns ± 0%  120ns ± 0%  ++15%",
			threshold: 5,
			want:      "  BenchmarkMalform-8 (15.00% slower)",
		},
		{
			name:      "line with malformed delta plus minus",
			line:      "BenchmarkMalform2-8  100  100ns ± 0%  120ns ± 0%  +-15%",
			threshold: 5,
			want:      "",
		},
		{
			name:      "empty line",
			line:      "",
			threshold: 5,
			want:      "",
		},
		{
			name:      "line with no benchmark data but with delta - still matches (regex is liberal)",
			line:      "some random text with +50%",
			threshold: 5,
			want:      "  some (50.00% slower)",
		},

		// Zero and boundary values
		{
			name:      "zero percent change",
			line:      "BenchmarkZero-8      100  100ns ± 0%  100ns ± 0%  +0.00%",
			threshold: 5,
			want:      "",
		},
		{
			name:      "very small positive delta below threshold",
			line:      "BenchmarkTiny-8      1000  100ns ± 0%  100.1ns ± 0%  +0.10%",
			threshold: 1,
			want:      "",
		},
		{
			name:      "decimal threshold matching",
			line:      "BenchmarkDecimal-8   500   1000ns ± 1%  1025ns ± 0%  +2.50%",
			threshold: 2.5,
			want:      "", // 2.50 is not > 2.5 (they're equal)
		},
		{
			name:      "decimal threshold above",
			line:      "BenchmarkDecimal2-8  500   1000ns ± 1%  1026ns ± 0%  +2.60%",
			threshold: 2.5,
			want:      "  BenchmarkDecimal2-8 (2.60% slower)",
		},

		// Multiple delta values in line (only first should be captured)
		{
			name:      "line with multiple percents (only first delta used)",
			line:      "BenchmarkMulti-8     100  50ns ± 5%  100ns ± 2%  +100.00%",
			threshold: 50,
			want:      "  BenchmarkMulti-8 (100.00% slower)",
		},

		// Very large regressions
		{
			name:      "extremely large regression",
			line:      "BenchmarkHuge-2      10   1000000ns ± 1%  50000000ns ± 0%  +4900.00%",
			threshold: 100,
			want:      "  BenchmarkHuge-2 (4900.00% slower)",
		},

		// Realistic benchstat output variations
		{
			name:      "typical benchstat line format with moderate regression",
			line:      "BenchmarkEncode-4    2000  500ns ± 1%   600ns ± 2%   +20.00%",
			threshold: 15,
			want:      "  BenchmarkEncode-4 (20.00% slower)",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := parseLine(testCase.line, testCase.threshold)
			if got != testCase.want {
				t.Errorf("parseLine(%q, %v) = %q, want %q", testCase.line, testCase.threshold, got, testCase.want)
			}
		})
	}
}

func Test_parseData_singlePackageSingleRegression(t *testing.T) {
	t.Parallel()

	input := `pkg: mypackage
goos: linux
goarch: amd64
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
BenchmarkFunc-8      100  101ns ± 1%  120ns ± 0%  +18.81%
BenchmarkOther-8     200  50ns ± 0%   45ns ± 0%   -10.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, osTxt, archTxt, cpuTxt := parseData(scanner, 5)

	// Check metadata extracted
	if osTxt != "linux" {
		t.Errorf("expected osTxt='linux', got %q", osTxt)
	}

	if archTxt != "amd64" {
		t.Errorf("expected archTxt='amd64', got %q", archTxt)
	}

	if cpuTxt != "Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz" {
		t.Errorf("expected cpuTxt with 'Core', got %q", cpuTxt)
	}

	// Check regression map
	if len(regMap) != 1 {
		t.Errorf("expected 1 package in regMap, got %d", len(regMap))
	}

	pkgRegs, exists := regMap["mypackage"]
	if !exists {
		t.Fatal("expected 'mypackage' in regMap")
	}

	if len(pkgRegs) != 1 {
		t.Errorf("expected 1 regression for 'mypackage', got %d: %v", len(pkgRegs), pkgRegs)
	}

	if !strings.Contains(pkgRegs[0], "BenchmarkFunc-8") {
		t.Errorf("expected regression to contain 'BenchmarkFunc-8', got %q", pkgRegs[0])
	}

	if !strings.Contains(pkgRegs[0], "18.81") {
		t.Errorf("expected regression to contain '18.81', got %q", pkgRegs[0])
	}
}

func Test_parseData_multiplePackages(t *testing.T) {
	t.Parallel()

	input := `pkg: package1
goos: darwin
goarch: arm64
cpu: Apple M1
BenchmarkParse-8     1000  1000ns ± 5%  1500ns ± 3%  +50.00%

pkg: package2
BenchmarkEncode-8    2000  500ns ± 1%   600ns ± 2%   +20.00%
BenchmarkDecode-8    1500  300ns ± 0%   250ns ± 1%   -16.67%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, osTxt, archTxt, cpuTxt := parseData(scanner, 15)

	// Verify metadata (only first occurrence captured)
	if osTxt != "darwin" {
		t.Errorf("expected osTxt='darwin', got %q", osTxt)
	}

	if archTxt != "arm64" {
		t.Errorf("expected archTxt='arm64', got %q", archTxt)
	}

	if cpuTxt != "Apple M1" {
		t.Errorf("expected cpuTxt='Apple M1', got %q", archTxt)
	}

	// Check both packages present
	if len(regMap) != 2 {
		t.Errorf("expected 2 packages in regMap, got %d: %v", len(regMap), regMap)
	}

	// package1 should have 1 regression (50% > 15)
	pkg1, exists := regMap["package1"]
	if !exists {
		t.Fatal("expected 'package1' in regMap")
	}

	if len(pkg1) != 1 {
		t.Errorf("expected 1 regression for 'package1', got %d", len(pkg1))
	}

	// package2 should have 1 regression (20% > 15, -16.67% is improvement and ignored)
	pkg2, exists := regMap["package2"]
	if !exists {
		t.Fatal("expected 'package2' in regMap")
	}

	if len(pkg2) != 1 {
		t.Errorf("expected 1 regression for 'package2', got %d: %v", len(pkg2), pkg2)
	}
}

func Test_parseData_metadataExtraction(t *testing.T) {
	t.Parallel()

	// Test that metadata is extracted only on first occurrence
	input := `goos: linux
goarch: amd64
cpu: Intel-First
goos: darwin
goarch: arm64
cpu: Apple-Second
pkg: testpkg`

	scanner := bufio.NewScanner(strings.NewReader(input))
	_, osTxt, archTxt, cpuTxt := parseData(scanner, 5)

	// Should use first occurrence of each
	if osTxt != "linux" {
		t.Errorf("expected first goos='linux', got %q", osTxt)
	}

	if archTxt != "amd64" {
		t.Errorf("expected first goarch='amd64', got %q", archTxt)
	}

	if cpuTxt != "Intel-First" {
		t.Errorf("expected first cpu='Intel-First', got %q", cpuTxt)
	}
}

func Test_parseData_mixedRegressionAndImprovements(t *testing.T) {
	t.Parallel()

	input := `pkg: mixedpkg
goos: linux
goarch: amd64
BenchmarkA-8    100  100ns  150ns  +50.00%
BenchmarkB-8    100  100ns  50ns   -50.00%
BenchmarkC-8    100  100ns  120ns  +20.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, _, _, _ := parseData(scanner, 25) //nolint:dogsled // We only care about regressions here

	regressions, exists := regMap["mixedpkg"]
	if !exists {
		t.Fatal("expected 'mixedpkg' in regMap")
	}

	// Only +50% should be included (both above 25 threshold and positive)
	// +20% is below 25 threshold, -50% is improvement and ignored
	if len(regressions) != 1 {
		t.Errorf("expected 1 regression above 25%% threshold, got %d: %v", len(regressions), regressions)
	}

	if !strings.Contains(regressions[0], "BenchmarkA-8") {
		t.Errorf("expected BenchmarkA-8 in regression, got %v", regressions)
	}
}

func Test_parseData_emptyInput(t *testing.T) {
	t.Parallel()

	input := ""
	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, osTxt, archTxt, cpuTxt := parseData(scanner, 5)

	if len(regMap) != 0 {
		t.Errorf("expected empty regMap for empty input, got %v", regMap)
	}

	if osTxt != "" {
		t.Errorf("expected empty osTxt, got %q", osTxt)
	}

	if archTxt != "" {
		t.Errorf("expected empty archTxt, got %q", archTxt)
	}

	if cpuTxt != "" {
		t.Errorf("expected empty cpuTxt, got %q", cpuTxt)
	}
}

func Test_parseData_metadataOnlyNoRegressions(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel`

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, osTxt, _, _ := parseData(scanner, 5)

	if len(regMap) != 0 {
		t.Errorf("expected no regressions, got %v", regMap)
	}

	if osTxt != "linux" {
		t.Errorf("expected metadata extracted, got osTxt=%q", osTxt)
	}
}

func Test_parseData_thresholdFiltering(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
BenchmarkA-8    100  100ns  105ns  +5.00%
BenchmarkB-8    100  100ns  115ns  +15.00%
BenchmarkC-8    100  100ns  125ns  +25.00%`

	// Test with threshold 10
	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, _, _, _ := parseData(scanner, 10) //nolint:dogsled // We only care about regressions here

	regressions := regMap["testpkg"]
	// Only +15% and +25% should be included (both > 10)
	if len(regressions) != 2 {
		t.Errorf("expected 2 regressions above 10%% threshold, got %d: %v", len(regressions), regressions)
	}

	// Verify specific benchmarks
	found := map[string]bool{}

	for _, reg := range regressions {
		if strings.Contains(reg, "BenchmarkB-8") {
			found["B"] = true
		}

		if strings.Contains(reg, "BenchmarkC-8") {
			found["C"] = true
		}
	}

	if !found["B"] || !found["C"] {
		t.Errorf("expected BenchmarkB and BenchmarkC in regressions, found %v", found)
	}
}

func Test_parseData_benchmarkBeforePkgHeader(t *testing.T) {
	t.Parallel()

	// Benchmark lines before any pkg: header should use empty string as package
	input := `goos: linux
BenchmarkOrphan-8    100  100ns  150ns  +50.00%
pkg: testpkg
BenchmarkNormal-8    100  100ns  150ns  +50.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, _, _, _ := parseData(scanner, 10) //nolint:dogsled // We only care about regressions here

	// Should have both empty string package and testpkg
	if len(regMap) != 2 {
		t.Errorf("expected 2 packages (empty and testpkg), got %d: %v", len(regMap), regMap)
	}

	orphan, hasOrphan := regMap["UNKNOWN"]
	if !hasOrphan || len(orphan) != 1 {
		t.Errorf("expected orphan regression in empty package")
	}

	testpkg, hasTestpkg := regMap["testpkg"]
	if !hasTestpkg || len(testpkg) != 1 {
		t.Errorf("expected regression in testpkg")
	}
}

func Test_parseData_multipleConsecutiveRegressions(t *testing.T) {
	t.Parallel()

	input := `pkg: multi
goos: linux
BenchmarkA-8    100  100ns  150ns  +50.00%
BenchmarkB-8    100  100ns  160ns  +60.00%
BenchmarkC-8    100  100ns  170ns  +70.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, _, _, _ := parseData(scanner, 10) //nolint:dogsled // We only care about regressions here

	regressions := regMap["multi"]
	if len(regressions) != 3 {
		t.Errorf("expected 3 regressions, got %d: %v", len(regressions), regressions)
	}

	// All should be in same package
	for i, reg := range regressions {
		if !strings.Contains(reg, "Benchmark") {
			t.Errorf("regression %d should be formatted correctly: %q", i, reg)
		}
	}
}

func Test_parseData_benchmarkName(t *testing.T) {
	t.Parallel()

	// Test that the benchmark name is correctly extracted from Fields()[0]
	tests := []struct {
		name      string
		line      string
		threshold float64
		wantName  string // First field of the line
	}{
		{
			name:      "simple name",
			line:      "BenchmarkFunc-8      100  100ns  150ns  +50%",
			threshold: 10,
			wantName:  "BenchmarkFunc-8",
		},
		{
			name:      "name with complex format",
			line:      "BenchmarkVeryLongName-128    1000  1000ns  2000ns  +100%",
			threshold: 50,
			wantName:  "BenchmarkVeryLongName-128",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := parseLine(testCase.line, testCase.threshold)
			if result != "" && !strings.Contains(result, testCase.wantName) {
				t.Errorf("parseLine result should contain benchmark name %q, got %q", testCase.wantName, result)
			}
		})
	}
}

func Test_processResults_withRegressions(t *testing.T) {
	t.Parallel()

	regMap := map[string][]string{
		"package1": {"  Bench1 (10.00% slower)", "  Bench2 (15.00% slower)"},
		"package2": {"  Bench3 (5.50% slower)"},
	}

	result := processResults(regMap, 5, "linux", "amd64", "Intel")
	if result != false {
		t.Errorf("processResults with regressions should return false, got %v", result)
	}
}

func Test_processResults_noRegressions(t *testing.T) {
	t.Parallel()

	regMap := map[string][]string{}

	result := processResults(regMap, 5, "linux", "amd64", "Intel")
	if result != true {
		t.Errorf("processResults without regressions should return true, got %v", result)
	}
}

func Test_processResults_emptyRegressionList(t *testing.T) {
	t.Parallel()

	// Map with empty lists should be treated as no regressions
	regMap := map[string][]string{
		"pkg1": {},
	}

	result := processResults(regMap, 5, "linux", "amd64", "Intel")
	// Since regMap length > 0, this should return false
	if result != false {
		t.Errorf("processResults with non-empty map should return false, got %v", result)
	}
}

func Test_Run_noRegressions(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  105ns  +5.00%
BenchmarkB-8    100  100ns  110ns  +10.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	result := Run(scanner, 15)

	if result != true {
		t.Errorf("Run() with all regressions below threshold should return true, got %v", result)
	}
}

func Test_Run_detectsRegressions(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  150ns  +50.00%
BenchmarkB-8    100  100ns  110ns  +10.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	result := Run(scanner, 20)

	if result != false {
		t.Errorf("Run() with regression above threshold should return false, got %v", result)
	}
}

func Test_Run_emptyInput(t *testing.T) {
	t.Parallel()

	input := ""
	scanner := bufio.NewScanner(strings.NewReader(input))
	result := Run(scanner, 5)

	if result != true {
		t.Errorf("Run() with empty input should return true, got %v", result)
	}
}

func Test_Run_metadataOnly(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel`

	scanner := bufio.NewScanner(strings.NewReader(input))
	result := Run(scanner, 5)

	if result != true {
		t.Errorf("Run() with metadata only should return true, got %v", result)
	}
}

func Test_Run_multiplePackages(t *testing.T) {
	t.Parallel()

	input := `pkg: pkg1
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  120ns  +20.00%

pkg: pkg2
BenchmarkB-8    100  100ns  115ns  +15.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	result := Run(scanner, 10)

	// Both have regressions above 10%, should return false
	if result != false {
		t.Errorf("Run() with multiple packages with regressions should return false, got %v", result)
	}
}

func Test_Run_onlyImprovements(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  50ns  -50.00%
BenchmarkB-8    100  100ns  80ns  -20.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	result := Run(scanner, 5)

	// Only improvements (negative deltas), no regressions
	if result != true {
		t.Errorf("Run() with only improvements should return true, got %v", result)
	}
}
