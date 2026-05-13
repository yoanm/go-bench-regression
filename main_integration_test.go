package benchreg_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	benchreg "github.com/yoanm/go-bench-regression"
)

func Test_Fixtures(t *testing.T) { //nolint:paralleltest // Can't be parallelized due to log output override !
	tests := []struct {
		name           string
		threshold      float64
		fixture        string
		expectedResult bool
		expectedOutput []string
	}{
		{
			name:           "base-case: 2 regressions above 10% - with 10% threshold",
			threshold:      10,
			fixture:        "base_case-2_regressions_above_10perc.txt",
			expectedResult: false,
			expectedOutput: []string{
				"❌  Performance regression detected — threshold: 10.0%",
				"🕵️Os \"linux\" — Arch \"amd64\" — CPU \"AMD EPYC 7763 64-Core Processor\"",
				"🗄️Package: github.com/yoanm/go-deps-diff/summary",
				"   🔎 sec/op",
				"      📈 GenerateForChanges-4 — 12.39% slower",
				"   🔎 allocs/op",
				"      📈 GenerateForChanges-4 — 11.75% slower",
			},
		},
		{
			name:           "base-case: 2 regressions above 10% - with 12% threshold",
			threshold:      12,
			fixture:        "base_case-2_regressions_above_10perc.txt",
			expectedResult: false,
			expectedOutput: []string{
				"❌  Performance regression detected — threshold: 12.0%",
				"🕵️Os \"linux\" — Arch \"amd64\" — CPU \"AMD EPYC 7763 64-Core Processor\"",
				"🗄️Package: github.com/yoanm/go-deps-diff/summary",
				"   🔎 sec/op",
				"      📈 GenerateForChanges-4 — 12.39% slower",
			},
		},
		{
			name:           "base-case: 9 regression above 10% with 3 geomean - with 10% threshold",
			threshold:      10,
			fixture:        "base_case-9_regressions_above_10perc_with_3_geomean.txt",
			expectedResult: false,
			expectedOutput: []string{
				"❌  Performance regression detected — threshold: 10.0%",
				"🕵️Os \"linux\" — Arch \"amd64\" — CPU \"AMD EPYC 7763 64-Core Processor\"",
				"🗄️Package: github.com/yoanm/go-deps-diff",
				"   🔎 sec/op",
				"      📈 Diff_ComposerDiff-4 — 38.10% slower",
				"   🔎 B/op",
				"      📈 Diff_ComposerDiff-4 — 52.24% slower",
				"   🔎 allocs/op",
				"      📈 Diff_ComposerDiff-4 — 36.77% slower",
				"🗄️Package: github.com/yoanm/go-deps-diff/managers/composer",
				"   🔎 sec/op",
				"      📈 BuildMapFromBytes-4 — 42.58% slower",
				"      📈 geomean — 11.93% slower",
				"   🔎 B/op",
				"      📈 BuildMapFromBytes-4 — 61.47% slower",
				"      📈 geomean — 17.32% slower",
				"   🔎 allocs/op",
				"      📈 BuildMapFromBytes-4 — 43.80% slower",
				"      📈 geomean — 12.87% slower",
			},
		},

		{
			name:           "base-case: 9 regression above 10% with 3 geomean - with 50% threshold",
			threshold:      50,
			fixture:        "base_case-9_regressions_above_10perc_with_3_geomean.txt",
			expectedResult: false,
			expectedOutput: []string{
				"❌  Performance regression detected — threshold: 50.0%",
				"🕵️Os \"linux\" — Arch \"amd64\" — CPU \"AMD EPYC 7763 64-Core Processor\"",
				"🗄️Package: github.com/yoanm/go-deps-diff",
				"   🔎 B/op",
				"      📈 Diff_ComposerDiff-4 — 52.24% slower",
				"🗄️Package: github.com/yoanm/go-deps-diff/managers/composer",
				"   🔎 B/op",
				"      📈 BuildMapFromBytes-4 — 61.47% slower",
			},
		},
		{
			name:           "base-case: regression below 1% - with 0.1% threshold",
			threshold:      0.1,
			fixture:        "base_case-regression_below_1perc.txt",
			expectedResult: false,
			expectedOutput: []string{
				"❌  Performance regression detected — threshold: 0.1%",
				"🕵️Os \"linux\" — Arch \"amd64\" — CPU \"AMD EPYC 7763 64-Core Processor\"",
				"🗄️Package: github.com/yoanm/go-deps-diff-summary",
				"   🔎 B/op",
				"      📈 GenerateForChanges-4 — 0.35% slower",
				"   🔎 allocs/op",
				"      📈 GenerateForChanges-4 — 0.24% slower",
			},
		},
		{
			name:           "base-case: regression below 1% - with 1% threshold",
			threshold:      1,
			fixture:        "base_case-regression_below_1perc.txt",
			expectedResult: true,
			expectedOutput: []string{
				"🎉 All good",
			},
		},
		{
			name:           "base-case: 4 regressions above 20% - with 20% threshold",
			threshold:      20,
			fixture:        "base_case_4_regression_above_20perc.txt",
			expectedResult: false,
			expectedOutput: []string{
				"❌  Performance regression detected — threshold: 20.0%",
				"🕵️Os \"linux\" — Arch \"amd64\" — CPU \"AMD EPYC 7763 64-Core Processor\"",
				"🗄️Package: github.com/yoanm/go-deps-diff",
				"   🔎 B/op",
				"      📈 Diff_ComposerDiff-4 — 28.86% slower",
				"   🔎 allocs/op",
				"      📈 Diff_ComposerDiff-4 — 34.91% slower",
				"🗄️Package: github.com/yoanm/go-deps-diff/managers/composer",
				"   🔎 B/op",
				"      📈 BuildMapFromBytes-4 — 28.32% slower",
				"   🔎 allocs/op",
				"      📈 BuildMapFromBytes-4 — 23.88% slower",
			},
		},
	}

	for _, testCase := range tests { //nolint:paralleltest // Can't be parallelized due to log output override !
		t.Run(testCase.name, func(t *testing.T) {
			// Load fixture files
			input, err := os.ReadFile("./testdata/" + testCase.fixture)
			if err != nil {
				t.Error(fmt.Errorf("reading requirement file = %w", err))

				return
			}

			stdoutReader, stdoutWriter, err := os.Pipe()
			if err != nil {
				t.Fatalf("creating fake stdout: %v", err)
			}

			defer func() {
				stdoutWriter.Close()
				stdoutReader.Close()
			}()

			scanner := bufio.NewScanner(bytes.NewReader(input))
			result := benchreg.Run(scanner, testCase.threshold, stdoutWriter)
			stdoutWriter.Close()

			if result != testCase.expectedResult {
				t.Errorf("expected first result %t, got %t", testCase.expectedResult, result)
			} else {
				validateOutput(t, stdoutReader, testCase.expectedOutput)
			}
		})
	}
}

func validateOutput(t *testing.T, stdoutReader *os.File, expected []string) {
	t.Helper()

	// Read output
	content, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Error(fmt.Errorf("reading stdout = %w", err))

		return
	}

	// Check expectations line by line
	testScanner := bufio.NewScanner(bytes.NewReader(content))

	expOutKey := 0
	for testScanner.Scan() {
		if strings.Contains(testScanner.Text(), expected[expOutKey]) {
			expOutKey++
			if expOutKey == len(expected) {
				break
			}
		}
	}

	if expOutKey != len(expected) {
		t.Errorf(
			"Unmatched expected output. Starting from expectation %d -> %q",
			expOutKey,
			expected[expOutKey],
		)
		t.Error("Actual output:\n" + string(content))
	}
}
