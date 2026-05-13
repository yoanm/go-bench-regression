// Package benchreg analyzes benchstat output and detects performance regressions
// above a specified threshold. It parses benchstat command output, extracts regression
// percentages from benchmark results, and reports those exceeding the threshold.
//
// Usage:
//
//	scanner := bufio.NewScanner(os.Stdin)
//	success := benchreg.Run(scanner, 10.0) // 10% threshold
//	if !success {
//		fmt.Println("Performance regressions detected")
//	}
package benchreg

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

const (
	pkgPrefix      = "pkg: "
	osPrefix       = "goos: "
	archPrefix     = "goarch: "
	cpuPrefix      = "cpu: "
	sectionKeyword = " vs base "

	unknownPackage  = "Unknown"
	unknownSection  = "Unknown section"
	unknownMetadata = "Unknown"
)

var (
	// Match lines like: "BenchmarkAbc-42  230ns  123ns  +90.00%".
	deltaRegex = regexp.MustCompile(`([+-]\d+\.?\d*)%`)
	// Match lines like: "│  allocs/op   │  allocs/op   vs base │".
	sectionRegex = regexp.MustCompile(`│\s+([^|\s]+)\s+vs base\s+│`)
)

// Run analyzes benchstat output from the provided scanner and detects performance
// regressions exceeding the specified threshold percentage.
//
// It parses benchstat output to extract:
//   - Package names and benchmark results
//   - Regression percentages (positive deltas)
//   - Metadata (operating system, architecture, CPU)
//
// Parameters:
//   - scanner: reads benchstat output line by line
//   - threshold: percentage threshold; regressions above this value are reported
//   - output: writer for the summary
//
// Returns:
//   - true if no regressions are detected (all deltas at or below threshold)
//   - false if regressions are detected and logged
//
// Example output format:
//
//	pkg: mypackage
//	goos: linux
//	goarch: amd64
//	cpu: Intel(R) Core(TM)
//	BenchmarkFunc-8    100  100ns  120ns  +20.00%
//
// With threshold 10%, the above example would trigger a regression report for the 20% delta.
func Run(input *bufio.Scanner, threshold float64, output io.StringWriter) bool {
	regMap, pkgOrder, sectionOrder, osTxt, archTxt, cpuTxt := parseData(input, threshold)

	if len(regMap) > 0 {
		printRegressions(output, regMap, pkgOrder, sectionOrder, threshold, osTxt, archTxt, cpuTxt)

		return false
	}

	if _, err := output.WriteString("🎉 All good\n"); err != nil {
		slog.Error("Error writing the output: " + err.Error())
	}

	return true
}

func parseData(input *bufio.Scanner, threshold float64) (
	map[string]map[string][]string, // Regressions list by packages and sections
	[]string, // Packages order (as read from the input)
	[]string, // Sections order (as read from the input)
	string, // os label
	string, // arch label
	string, // cpu label
) {
	regMap := map[string]map[string][]string{}
	currentPkg := unknownPackage     // Default value in case of no package header !
	currentSection := unknownSection // Default value in case of no section header !

	var (
		pkgOrder     []string
		sectionOrder []string
	)

	osTxt := unknownMetadata
	archTxt := unknownMetadata
	cpuTxt := unknownMetadata

	sectionMap := map[string]struct{}{}

	for input.Scan() {
		line := input.Text()
		// Check if lines are related to package, section, os, arch or cpu info
		if detectNonBenchLine(line, &currentPkg, &currentSection, &osTxt, &archTxt, &cpuTxt) {
			continue // No need to parse the line further, we have what we want
		}

		if txt := parseLine(line, threshold); txt != "" {
			if regMap[currentPkg] == nil {
				regMap[currentPkg] = map[string][]string{}
				pkgOrder = append(pkgOrder, currentPkg)
			}

			if _, exist := sectionMap[currentSection]; !exist {
				sectionOrder = append(sectionOrder, currentSection)
				sectionMap[currentSection] = struct{}{}
			}

			regMap[currentPkg][currentSection] = append(regMap[currentPkg][currentSection], txt)
		}
	}

	return regMap, pkgOrder, sectionOrder, osTxt, archTxt, cpuTxt
}

func detectNonBenchLine(
	line string,
	currentPkg *string,
	currentSection *string,
	osTxt *string,
	archTxt *string,
	cpuTxt *string,
) bool {
	switch {
	case strings.HasPrefix(line, pkgPrefix):
		*currentPkg = strings.TrimSpace(strings.TrimPrefix(line, pkgPrefix))

		return true
	case strings.Contains(line, sectionKeyword):
		if matches := sectionRegex.FindStringSubmatch(line); len(matches) > 1 {
			*currentSection = matches[1]

			return true
		}

		*currentSection = unknownSection // Reset
	case *osTxt == unknownMetadata && strings.HasPrefix(line, osPrefix):
		*osTxt = strings.TrimSpace(strings.TrimPrefix(line, osPrefix))

		return true
	case *archTxt == unknownMetadata && strings.HasPrefix(line, archPrefix):
		*archTxt = strings.TrimSpace(strings.TrimPrefix(line, archPrefix))

		return true
	case *cpuTxt == unknownMetadata && strings.HasPrefix(line, cpuPrefix):
		*cpuTxt = strings.TrimSpace(strings.TrimPrefix(line, cpuPrefix))

		return true
	}

	return false
}

func parseLine(line string, threshold float64) string {
	if matches := deltaRegex.FindStringSubmatch(line); len(matches) > 1 {
		delta, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			slog.Warn("Error parsing delta. Skipping line", "line", line)

			return ""
		}

		// Positive delta means regression (slower)
		if delta > threshold {
			return fmt.Sprintf("%s — %.2f%% slower", strings.Fields(line)[0], delta)
		}
	}

	return ""
}

func printRegressions(
	output io.StringWriter,
	regMap map[string]map[string][]string,
	pkgOrder []string,
	sectionOrder []string,
	threshold float64,
	osTxt string,
	archTxt string,
	cpuTxt string,
) {
	txt := strings.Builder{}
	fmt.Fprintf(&txt, "❌  Performance regression detected — threshold: %.1f%%\n", threshold)
	fmt.Fprintf(&txt, "🕵️Os %q — Arch %q — CPU %q\n", osTxt, archTxt, cpuTxt)

	// Sort packages and section in order to have a deterministic output (way easier for tests)
	inOrderMapIteratorHelper(regMap, pkgOrder, func(pkg string, subBegMap map[string][]string) {
		txt.WriteString("🗄️Package: " + pkg + "\n")
		inOrderMapIteratorHelper(subBegMap, sectionOrder, func(section string, regList []string) {
			txt.WriteString("   ➖  " + section + "\n")

			for _, reg := range regList {
				txt.WriteString("      📈 " + reg + "\n")
			}
		})
	})

	if _, err := output.WriteString(txt.String()); err != nil {
		slog.Error("Error writing the output: " + err.Error())
	}
}

func inOrderMapIteratorHelper[K comparable, V any](theMap map[K]V, orderList []K, processor func(key K, val V)) {
	for _, key := range orderList {
		val, exists := theMap[key]
		if !exists {
			continue
		}

		processor(key, val)
	}
}
