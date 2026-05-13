package benchreg_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	benchreg "github.com/yoanm/go-bench-regression"
)

func Test_Run_noRegressions(t *testing.T) {
	t.Parallel()

	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  105ns  +5.00%
BenchmarkB-8    100  100ns  110ns  +10.00%`

	scanner := bufio.NewScanner(strings.NewReader(input))
	result := benchreg.Run(scanner, 15, os.NewFile(0, os.DevNull))

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
	result := benchreg.Run(scanner, 20, os.NewFile(0, os.DevNull))

	if result != false {
		t.Errorf("Run() with regression above threshold should return false, got %v", result)
	}
}

func Test_Run_emptyInput(t *testing.T) {
	t.Parallel()

	input := ""
	scanner := bufio.NewScanner(strings.NewReader(input))
	result := benchreg.Run(scanner, 5, os.NewFile(0, os.DevNull))

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
	result := benchreg.Run(scanner, 5, os.NewFile(0, os.DevNull))

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
	result := benchreg.Run(scanner, 10, os.NewFile(0, os.DevNull))

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
	result := benchreg.Run(scanner, 5, os.NewFile(0, os.DevNull))

	// Only improvements (negative deltas), no regressions
	if result != true {
		t.Errorf("Run() with only improvements should return true, got %v", result)
	}
}
