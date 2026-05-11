package benchreg_test

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
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
				"ERROR Performance regression detected (threshold: 10.0%):",
				"ERROR Os \"linux\" / Arch \"amd64\" / CPU \"AMD EPYC 7763 64-Core Processor\"",
				"ERROR Package: github.com/yoanm/go-deps-diff/summary",
				"ERROR   sec/op",
				"ERROR     GenerateForChanges-4 (12.39% slower)",
				"ERROR   allocs/op",
				"ERROR     GenerateForChanges-4 (11.75% slower)",
			},
		},
		{
			name:           "base-case: 2 regressions above 10% - with 12% threshold",
			threshold:      12,
			fixture:        "base_case-2_regressions_above_10perc.txt",
			expectedResult: false,
			expectedOutput: []string{
				"ERROR Performance regression detected (threshold: 12.0%):",
				"ERROR Os \"linux\" / Arch \"amd64\" / CPU \"AMD EPYC 7763 64-Core Processor\"",
				"ERROR Package: github.com/yoanm/go-deps-diff/summary",
				"ERROR   sec/op",
				"ERROR     GenerateForChanges-4 (12.39% slower)",
			},
		},
		{
			name:           "base-case: 9 regression above 10% with 3 geomean - with 10% threshold",
			threshold:      10,
			fixture:        "base_case-9_regressions_above_10perc_with_3_geomean.txt",
			expectedResult: false,
			expectedOutput: []string{
				"ERROR Performance regression detected (threshold: 10.0%):",
				"ERROR Os \"linux\" / Arch \"amd64\" / CPU \"AMD EPYC 7763 64-Core Processor\"",
				"ERROR Package: github.com/yoanm/go-deps-diff",
				"ERROR   sec/op",
				"ERROR     Diff_ComposerDiff-4 (38.10% slower)",
				"ERROR   B/op",
				"ERROR     Diff_ComposerDiff-4 (52.24% slower)",
				"ERROR   allocs/op",
				"ERROR     Diff_ComposerDiff-4 (36.77% slower)",
				"ERROR Package: github.com/yoanm/go-deps-diff/managers/composer",
				"ERROR   sec/op",
				"ERROR     BuildMapFromBytes-4 (42.58% slower)",
				"ERROR     geomean (11.93% slower)",
				"ERROR   B/op",
				"ERROR     BuildMapFromBytes-4 (61.47% slower)",
				"ERROR     geomean (17.32% slower)",
				"ERROR   allocs/op",
				"ERROR     BuildMapFromBytes-4 (43.80% slower)",
				"ERROR     geomean (12.87% slower)",
			},
		},
		{
			name:           "base-case: regression below 1% - with 0.1% threshold",
			threshold:      0.1,
			fixture:        "base_case-regression_below_1perc.txt",
			expectedResult: false,
			expectedOutput: []string{
				"ERROR Performance regression detected (threshold: 0.1%):",
				"ERROR Os \"linux\" / Arch \"amd64\" / CPU \"AMD EPYC 7763 64-Core Processor\"",
				"ERROR Package: github.com/yoanm/go-deps-diff-summary",
				"ERROR   B/op",
				"ERROR     GenerateForChanges-4 (0.35% slower)",
				"ERROR   allocs/op",
				"ERROR     GenerateForChanges-4 (0.24% slower)",
			},
		},
		{
			name:           "base-case: 4 regressions above 20% - with 20% threshold",
			threshold:      20,
			fixture:        "base_case_4_regression_above_20perc.txt",
			expectedResult: false,
			expectedOutput: []string{
				"ERROR Performance regression detected (threshold: 20.0%):",
				"ERROR Os \"linux\" / Arch \"amd64\" / CPU \"AMD EPYC 7763 64-Core Processor\"",
				"ERROR Package: github.com/yoanm/go-deps-diff",
				"ERROR   B/op",
				"ERROR     Diff_ComposerDiff-4 (28.86% slower)",
				"ERROR   allocs/op",
				"ERROR     Diff_ComposerDiff-4 (34.91% slower)",
				"ERROR Package: github.com/yoanm/go-deps-diff/managers/composer",
				"ERROR   B/op",
				"ERROR     BuildMapFromBytes-4 (28.32% slower)",
				"ERROR   allocs/op",
				"ERROR     BuildMapFromBytes-4 (23.88% slower)",
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

			var buf bytes.Buffer
			log.SetOutput(&buf) // Capture log output

			scanner := bufio.NewScanner(bytes.NewReader(input))
			result := benchreg.Run(scanner, testCase.threshold)

			if result != testCase.expectedResult {
				t.Errorf("expected first result %t, got %t", testCase.expectedResult, result)
			} else {
				out := buf.String()
				testScanner := bufio.NewScanner(&buf)

				expOutKey := 0
				for testScanner.Scan() {
					if strings.Contains(testScanner.Text(), testCase.expectedOutput[expOutKey]) {
						expOutKey++
					}
				}

				if expOutKey != len(testCase.expectedOutput) {
					t.Errorf(
						"Unmatched expected output. Starting from expectation %d -> %q",
						expOutKey,
						testCase.expectedOutput[expOutKey],
					)
					t.Error("Actual output:\n" + out)
				}
			}
		})
	}
}
