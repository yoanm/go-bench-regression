//nolint:paralleltest // Can't be parallelized due to stdIn/args override !
package main

import (
	"log"
	"os"
	"slices"
	"testing"
)

// TestCLI_noArguments tests CLI behavior when no arguments are provided.
func TestCLI_noArguments(t *testing.T) {
	closeFn := setupCLITest(t, []string{}, nil)
	defer closeFn()

	if exitCode := execute(); exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

// TestCLI_invalidThresholdFormat tests CLI with non-numeric threshold.
func TestCLI_invalidThresholdFormat(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
	}{
		{"non-numeric", "abc"},
		{"malformed float", "10.50.5"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			args := []string{testCase.threshold}
			input := ""

			closeFn := setupCLITest(t, args, &input)
			defer closeFn()

			if exitCode := execute(); exitCode != 3 {
				t.Errorf("threshold %q: expected exit code 3, got %d", testCase.threshold, exitCode)
			}
		})
	}
}

// TestCLI_outOfRangeThreshold tests thresholds outside valid range (>0).
func TestCLI_outOfRangeThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
	}{
		{"zero", "0"},
		{"negative", "-10"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			args := []string{testCase.threshold}
			input := ""

			closeFn := setupCLITest(t, args, &input)
			defer closeFn()

			if exitCode := execute(); exitCode != 3 {
				t.Errorf("threshold %s: expected exit code 3, got %d", testCase.threshold, exitCode)
			}
		})
	}
}

// TestCLI_validThresholds tests valid threshold values.
func TestCLI_validThresholds(t *testing.T) {
	tests := []string{"1", "50.5", "99", "100", "0.23", "200"}

	for _, threshold := range tests {
		t.Run("threshold_"+threshold, func(t *testing.T) {
			args := []string{threshold}

			closeFn := setupCLITest(t, args, nil)
			defer closeFn()

			// Exit code should be 3 (no input from terminal), not 1 or 2 (arg errors)
			if exitCode := execute(); exitCode != 4 {
				t.Errorf("threshold %s: expected exit code 4 (no input), got %d", threshold, exitCode)
			}
		})
	}
}

// TestCLI_emptyPipedInput tests when empty benchstat is piped.
func TestCLI_emptyPipedInput(t *testing.T) {
	args := []string{"5"}
	input := ""

	closeFn := setupCLITest(t, args, &input)
	defer closeFn()

	if code := execute(); code != 0 {
		t.Errorf("expected exit code 0 for empty input, got %d", code)
	}
}

// TestCLI_noRegressions tests benchstat output with no regressions.
func TestCLI_noRegressions(t *testing.T) {
	args := []string{"20"}
	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  105ns  +5.00%
BenchmarkB-8    100  100ns  110ns  +10.00%`

	closeFn := setupCLITest(t, args, &input)
	defer closeFn()

	if code := execute(); code != 0 {
		t.Errorf("expected exit code 0 (no regressions), got %d", code)
	}
}

// TestCLI_withRegressions tests benchstat output with regressions.
func TestCLI_withRegressions(t *testing.T) {
	args := []string{"20"}
	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  150ns  +50.00%
BenchmarkB-8    100  100ns  110ns  +10.00%`

	closeFn := setupCLITest(t, args, &input)
	defer closeFn()

	if code := execute(); code != 1 {
		t.Errorf("expected exit code 1 (regressions found), got %d", code)
	}
}

// TestCLI_realWorldExample tests with realistic benchstat output.
func TestCLI_realWorldExample(t *testing.T) {
	args := []string{"15"}
	input := `goos: linux
goarch: amd64
pkg: github.com/example/lib
cpu: Intel(R) Core(TM) i7-8700K
BenchmarkParse-8             1000  1000ns ± 5%  1500ns ± 3%  +50.00%
BenchmarkMarshal-8            500  2000ns ± 2%  1800ns ± 1%  -10.00%
BenchmarkEncode-8            2000  500ns ± 1%   600ns ± 2%   +20.00%

pkg: github.com/example/lib/internal
BenchmarkDecode-8            1500  300ns ± 0%   250ns ± 1%   -16.67%`

	closeFn := setupCLITest(t, args, &input)
	defer closeFn()

	if code := execute(); code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

// TestCLI_onlyImprovements tests benchstat with only improvements.
func TestCLI_onlyImprovements(t *testing.T) {
	args := []string{"5"}
	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel
BenchmarkA-8    100  100ns  50ns   -50.00%
BenchmarkB-8    100  100ns  80ns   -20.00%`

	closeFn := setupCLITest(t, args, &input)
	defer closeFn()

	if code := execute(); code != 0 {
		t.Errorf("expected exit code 0 (no regressions), got %d", code)
	}
}

// TestCLI_exactThresholdMatch tests when delta exactly matches threshold.
func TestCLI_exactThresholdMatch(t *testing.T) {
	args := []string{"10"}
	input := `pkg: testpkg
goos: linux
BenchmarkA-8    100  100ns  110ns  +10.00%`

	closeFn := setupCLITest(t, args, &input)
	defer closeFn()

	if code := execute(); code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

// TestCLI_justAboveThreshold tests when delta is just above threshold.
func TestCLI_justAboveThreshold(t *testing.T) {
	args := []string{"10"}
	input := `pkg: testpkg
goos: linux
BenchmarkA-8    100  100ns  110.1ns  +10.10%`

	closeFn := setupCLITest(t, args, &input)
	defer closeFn()

	if code := execute(); code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func setupCLITest(t *testing.T, args []string, input *string) func() {
	t.Helper()
	log.SetOutput(os.NewFile(0, os.DevNull)) // Avoid logs output mixed with test output

	oldStdOut, oldStdErr := os.Stdout, os.Stderr

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating fake stdout: %v", err)
	}

	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating fake stderr: %v", err)
	}

	os.Stdout, os.Stderr = stdoutWriter, stderrWriter

	oldArgs, oldStdIn := os.Args, os.Stdin

	os.Args = slices.Concat([]string{"fake-cmd-name"}, args)

	var stdinReader *os.File

	if input != nil {
		var stdinWriter *os.File

		stdinReader, stdinWriter, err = os.Pipe()
		if err != nil {
			t.Fatalf("creating fake stdin: %v", err)
		}

		os.Stdin = stdinReader

		if _, err = stdinWriter.WriteString(*input); err != nil {
			t.Fatalf("writing to fake stdin: %v", err)
		}

		stdinWriter.Close() // Close it else program run forever waiting for new data
	}

	return func() {
		// Revert changes at the end !
		os.Args, os.Stdin = oldArgs, oldStdIn

		if nil != stdinReader {
			stdinReader.Close()
		}

		os.Stdout, os.Stderr = oldStdOut, oldStdErr

		stdoutWriter.Close()
		stdoutReader.Close()
		stderrWriter.Close()
		stderrReader.Close()
	}
}
