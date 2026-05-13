package benchreg

import (
	"bufio"
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/andreyvit/diff"
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
			want:      "BenchmarkFunc-8 — 18.81% slower",
		},
		{
			name:      "large regression",
			line:      "BenchmarkLarge-4     50   5000ns ± 2%  10000ns ± 1%  +100.00%",
			threshold: 10,
			want:      "BenchmarkLarge-4 — 100.00% slower",
		},
		{
			name:      "small regression above threshold",
			line:      "BenchmarkSmall-8     1000  100ns ± 0%  105.5ns ± 0%  +5.50%",
			threshold: 5,
			want:      "BenchmarkSmall-8 — 5.50% slower",
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
			want:      "BenchmarkAbove-8 — 10.05% slower",
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
			want:      "BenchmarkMalform-8 — 15.00% slower",
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
			want:      "some — 50.00% slower",
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
			want:      "BenchmarkDecimal2-8 — 2.60% slower",
		},

		// Multiple delta values in line (only first should be captured)
		{
			name:      "line with multiple percents (only first delta used)",
			line:      "BenchmarkMulti-8     100  50ns ± 5%  100ns ± 2%  +100.00%",
			threshold: 50,
			want:      "BenchmarkMulti-8 — 100.00% slower",
		},

		// Very large regressions
		{
			name:      "extremely large regression",
			line:      "BenchmarkHuge-2      10   1000000ns ± 1%  50000000ns ± 0%  +4900.00%",
			threshold: 100,
			want:      "BenchmarkHuge-2 — 4900.00% slower",
		},

		// Realistic benchstat output variations
		{
			name:      "typical benchstat line format with moderate regression",
			line:      "BenchmarkEncode-4    2000  500ns ± 1%   600ns ± 2%   +20.00%",
			threshold: 15,
			want:      "BenchmarkEncode-4 — 20.00% slower",
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

	expectedPkgName := "mypackage"
	expectedPkgOrder := []string{expectedPkgName}
	expectedSectionOrder := []string{unknownSection}

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, pkgOrder, sectionOrder, osTxt, archTxt, cpuTxt := parseData(scanner, 5)

	// Check metadata extracted
	if !slices.Equal(pkgOrder, expectedPkgOrder) {
		t.Errorf("expected pkgOrder=%v, got %v", expectedPkgOrder, pkgOrder)
	}

	if !slices.Equal(sectionOrder, expectedSectionOrder) {
		t.Errorf("expected sectionOrder=%v, got %v", expectedSectionOrder, sectionOrder)
	}

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

	pkgRegs, pkgExists := regMap[expectedPkgName]
	if !pkgExists {
		t.Fatal("expected '" + expectedPkgName + "' package in regMap")
	}

	sectionRegs, sectionExists := pkgRegs[unknownSection]
	if !sectionExists {
		t.Fatal("expected '" + unknownSection + "' section in pkgRegs")
	}

	if len(sectionRegs) != 1 {
		t.Errorf("expected 1 regression for 'mypackage', got %d: %v", len(sectionRegs), sectionRegs)
	}

	if !strings.Contains(sectionRegs[0], "BenchmarkFunc-8") {
		t.Errorf("expected regression to contain 'BenchmarkFunc-8', got %q", sectionRegs[0])
	}

	if !strings.Contains(sectionRegs[0], "18.81") {
		t.Errorf("expected regression to contain '18.81', got %q", sectionRegs[0])
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

	expectedPkg1Name := "package1"
	expectedPkg2Name := "package2"
	expectedPkgOrder := []string{expectedPkg1Name, expectedPkg2Name}
	expectedSectionOrder := []string{unknownSection}

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, pkgOrder, sectionOrder, osTxt, archTxt, cpuTxt := parseData(scanner, 15)

	// Verify metadata (only first occurrence captured)
	if !slices.Equal(pkgOrder, expectedPkgOrder) {
		t.Errorf("expected pkgOrder=%v, got %v", expectedPkgOrder, pkgOrder)
	}

	if !slices.Equal(sectionOrder, expectedSectionOrder) {
		t.Errorf("expected sectionOrder=%v, got %v", expectedSectionOrder, sectionOrder)
	}

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
	pkgRegs, exists := regMap[expectedPkg1Name]
	if !exists {
		t.Fatal("expected '" + expectedPkg1Name + "' in regMap")
	}

	sectionRegs, sectionExists := pkgRegs[unknownSection]
	if !sectionExists {
		t.Fatal("expected '" + unknownSection + "' section in pkgRegs")
	}

	if len(sectionRegs) != 1 {
		t.Errorf("expected 1 regression for '"+expectedPkg1Name+"', got %d", len(sectionRegs))
	}

	// package2 should have 1 regression (20% > 15, -16.67% is improvement and ignored)
	pkg2Regs, exists2 := regMap[expectedPkg2Name]
	if !exists2 {
		t.Fatal("expected '" + expectedPkg2Name + "' in regMap")
	}

	section2Regs, sectionExists2 := pkg2Regs[unknownSection]
	if !sectionExists2 {
		t.Fatal("expected '" + unknownSection + "' section in pkgRegs")
	}

	if len(section2Regs) != 1 {
		t.Errorf("expected 1 regression for '"+expectedPkg2Name+"', got %d: %v", len(section2Regs), section2Regs)
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
	_, _, _, osTxt, archTxt, cpuTxt := parseData(scanner, 5) //nolint:dogsled // Focused on os/arch/cpu only

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
	regMap, _, _, _, _, _ := parseData(scanner, 25) //nolint:dogsled // We only care about regressions here

	pkgRegs, exists := regMap["mixedpkg"]
	if !exists {
		t.Fatal("expected 'mixedpkg' in regMap")
	}

	regressions, sectionExists := pkgRegs[unknownSection]
	if !sectionExists {
		t.Fatal("expected '" + unknownSection + "' section in pkgRegs")
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
	regMap, pkgOrder, sectionOrder, osTxt, archTxt, cpuTxt := parseData(scanner, 5)

	if len(regMap) != 0 {
		t.Errorf("expected empty regMap for empty input, got %v", regMap)
	}

	if len(pkgOrder) != 0 {
		t.Errorf("expected empty pkgOrder, got %v", pkgOrder)
	}

	if len(sectionOrder) != 0 {
		t.Errorf("expected empty sectionOrder, got %v", sectionOrder)
	}

	if osTxt != unknownMetadata {
		t.Errorf("expected osTxt=%q, got %q", unknownMetadata, osTxt)
	}

	if archTxt != unknownMetadata {
		t.Errorf("expected archTxt=%q, got %q", unknownMetadata, archTxt)
	}

	if cpuTxt != unknownMetadata {
		t.Errorf("expected cpuTxt=%q, got %q", unknownMetadata, cpuTxt)
	}
}

func Test_parseData_metadataOnlyNoRegressions(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel`

	scanner := bufio.NewScanner(strings.NewReader(input))
	regMap, _, _, osTxt, archTxt, cpuTxt := parseData(scanner, 5)

	if len(regMap) != 0 {
		t.Errorf("expected no regressions, got %v", regMap)
	}

	if osTxt != "linux" {
		t.Errorf("expected metadata extracted, got osTxt=%q", osTxt)
	}

	if archTxt != "amd64" {
		t.Errorf("expected metadata extracted, got archTxt=%q", archTxt)
	}

	if cpuTxt != "Intel" {
		t.Errorf("expected metadata extracted, got cpuTxt=%q", cpuTxt)
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
	regMap, _, _, _, _, _ := parseData(scanner, 10) //nolint:dogsled // We only care about regressions here

	regressions := regMap["testpkg"][unknownSection]
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
	regMap, _, _, _, _, _ := parseData(scanner, 10) //nolint:dogsled // We only care about regressions here

	// Should have both empty string package and testpkg
	if len(regMap) != 2 {
		t.Errorf("expected 2 packages (empty and testpkg), got %d: %v", len(regMap), regMap)
	}

	orphan, hasOrphan := regMap[unknownPackage]
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
	regMap, _, _, _, _, _ := parseData(scanner, 10) //nolint:dogsled // We only care about regressions here

	regressions := regMap["multi"][unknownSection]
	if len(regressions) != 3 {
		t.Errorf("expected 3 regressions, got %d: %v", len(regressions), regressions)
	}

	// All should be in same package
	for idx, reg := range regressions {
		if !strings.Contains(reg, "Benchmark") {
			t.Errorf("regression %d should be formatted correctly: %q", idx, reg)
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

func Test_printRegressions(t *testing.T) {
	t.Parallel()

	regMap := map[string]map[string][]string{
		"package1": {
			unknownSection: {" Bench1 — 10.00% slower", " Bench2 — 15.00% slower"},
		},
		"package2": {
			unknownSection: {" Bench3 — 5.50% slower"},
		},
	}
	pkgOrder := []string{"package1", "package2"}
	sectionOrder := []string{unknownSection}

	expected := `❌  Performance regression detected — threshold: 5.0%
🕵️Os "linux" — Arch "amd64" — CPU "Intel"
🗄️Package: package1
   🔎 Unknown section
      📈  Bench1 — 10.00% slower
      📈  Bench2 — 15.00% slower
🗄️Package: package2
   🔎 Unknown section
      📈  Bench3 — 5.50% slower
`

	output := &bytes.Buffer{}

	printRegressions(output, regMap, pkgOrder, sectionOrder, 5, "linux", "amd64", "Intel")

	content, err := io.ReadAll(output)
	if err != nil {
		t.Error("Unable to read from output writer: " + err.Error())
	} else if string(content) != expected {
		t.Errorf("Unexpected output, diff: %s", diff.LineDiff(expected, string(content)))
	}
}

// Test_parseLine_geomeanLine tests parsing of geomean lines to ensure they don't
// create spurious regressions. Geomean lines appear in benchstat output but should
// be handled appropriately.
func Test_parseLine_geomeanLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		line      string
		threshold float64
		want      string
	}{
		{
			name:      "geomean line with delta (should be treated like any other line)",
			line:      "geomean                                                      +15.50%",
			threshold: 5,
			want:      "geomean — 15.50% slower", // Geomean is just a benchmark name in the regex
		},
		{
			name:      "geomean line with delta below threshold",
			line:      "geomean                                                      +3.00%",
			threshold: 5,
			want:      "", // Below threshold, no regression
		},
		{
			name:      "geomean line with delta exactly at threshold",
			line:      "geomean                                                      +10.00%",
			threshold: 10,
			want:      "", // Exact threshold doesn't match (> not >=)
		},
		{
			name:      "geomean line with improvement",
			line:      "geomean                                                      -5.00%",
			threshold: 5,
			want:      "", // Negative delta, no regression
		},
		{
			name:      "geomean line with no delta",
			line:      "geomean",
			threshold: 5,
			want:      "", // No delta, no regression
		},
	}

	for _, testCase := range tests {
		// Capture for parallel execution
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := parseLine(testCase.line, testCase.threshold)
			if result != testCase.want {
				t.Errorf("parseLine(%q, %v) = %q, want %q", testCase.line, testCase.threshold, result, testCase.want)
			}
		})
	}
}

// Test_parseData_variableSpacing tests robustness of parseData when benchstat output
// has inconsistent whitespace (multiple spaces, tabs, leading/trailing spaces).
func Test_parseData_variableSpacing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		threshold  float64
		wantPkg    string
		wantHasReg bool
	}{
		{
			name: "single spaces (standard benchstat format)",
			input: `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "multiple spaces between fields",
			input: `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkTest-8     100    100ns    120ns    +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "tabs instead of spaces",
			input: `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkTest-8	100	100ns	120ns	+20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "mixed spaces and tabs",
			input: `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkTest-8  	 100  	100ns  	 120ns  	+20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "leading spaces on benchmark line",
			input: `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
   BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "no regressions with variable spacing",
			input: `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkTest-8  	 100  	100ns  	 102ns  	+2.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: false,
		},
	}

	for _, testCase := range tests {
		// Capture for parallel execution
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scanner := bufio.NewScanner(strings.NewReader(testCase.input))
			regMap, _, _, _, _, _ := parseData(scanner, testCase.threshold)

			hasRegressions := len(regMap[testCase.wantPkg]) > 0
			if hasRegressions != testCase.wantHasReg {
				t.Errorf("parseData() found regressions=%v, want %v. regMap=%+v", hasRegressions, testCase.wantHasReg, regMap)
			}
		})
	}
}

// Test_parseData_packageNamePatterns tests parsing with real Go package naming patterns.
// Verifies that complex package names (github.com/org/repo, internal/pkg_name, etc.) are
// handled correctly.
func Test_parseData_packageNamePatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		threshold  float64
		wantPkg    string
		wantHasReg bool
	}{
		{
			name: "github.com/org/repo package name",
			input: `pkg: github.com/org/repo
goos: linux
goarch: amd64
cpu: Intel
BenchmarkFunc-8     100  100ns  130ns  +30.00%`,
			threshold:  10,
			wantPkg:    "github.com/org/repo",
			wantHasReg: true,
		},
		{
			name: "internal package name",
			input: `pkg: internal/helpers
goos: linux
goarch: amd64
cpu: Intel
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "internal/helpers",
			wantHasReg: true,
		},
		{
			name: "package with underscores",
			input: `pkg: my_pkg_name
goos: linux
goarch: amd64
cpu: Intel
BenchmarkFunc-8     100  100ns  115ns  +15.00%`,
			threshold:  10,
			wantPkg:    "my_pkg_name",
			wantHasReg: true,
		},
		{
			name: "package with hyphens (go module naming)",
			input: `pkg: github.com/my-org/my-repo
goos: linux
goarch: amd64
cpu: Intel
BenchmarkTest-8     100  100ns  125ns  +25.00%`,
			threshold:  10,
			wantPkg:    "github.com/my-org/my-repo",
			wantHasReg: true,
		},
		{
			name: "nested internal package",
			input: `pkg: internal/pkg/helpers/util
goos: linux
goarch: amd64
cpu: Intel
BenchmarkFunc-8     100  100ns  110ns  +10.00%`,
			threshold:  5,
			wantPkg:    "internal/pkg/helpers/util",
			wantHasReg: true,
		},
	}

	for _, testCase := range tests {
		// Capture for parallel execution
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scanner := bufio.NewScanner(strings.NewReader(testCase.input))
			regMap, _, _, _, _, _ := parseData(scanner, testCase.threshold)

			hasRegressions := len(regMap[testCase.wantPkg]) > 0
			if hasRegressions != testCase.wantHasReg {
				t.Errorf("parseData() for package %q found regressions=%v, want %v. regMap=%+v", testCase.wantPkg, hasRegressions, testCase.wantHasReg, regMap)
			}
		})
	}
}

// Test_parseData_metadataVariations tests parseData robustness with metadata field variations
// such as missing fields, reordered fields, and extra whitespace.
func Test_parseData_metadataVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		threshold  float64
		wantPkg    string
		wantHasReg bool
	}{
		{
			name: "missing goarch field",
			input: `pkg: testpkg
goos: linux
cpu: Intel
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "missing goos field",
			input: `pkg: testpkg
goarch: amd64
cpu: Intel
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "missing cpu field",
			input: `pkg: testpkg
goos: linux
goarch: amd64
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "reordered metadata fields",
			input: `pkg: testpkg
cpu: Intel
goarch: amd64
goos: linux
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "extra whitespace in metadata values",
			input: `pkg: testpkg
goos: linux 
goarch: amd64  
cpu: Intel Core i7
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
		{
			name: "all metadata fields missing except pkg",
			input: `pkg: testpkg
BenchmarkTest-8     100  100ns  120ns  +20.00%`,
			threshold:  5,
			wantPkg:    "testpkg",
			wantHasReg: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scanner := bufio.NewScanner(strings.NewReader(testCase.input))
			regMap, _, _, _, _, _ := parseData(scanner, testCase.threshold)

			hasRegressions := len(regMap[testCase.wantPkg]) > 0
			if hasRegressions != testCase.wantHasReg {
				t.Errorf("parseData() found regressions=%v, want %v. regMap=%+v", hasRegressions, testCase.wantHasReg, regMap)
			}
		})
	}
}
