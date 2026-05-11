package benchreg

import (
	"bufio"
	"fmt"
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

	unknownPackage = "Unknown section"
	unknownSection = "Unknown package"
)

// Match lines like: "BenchmarkAbc-42  230ns  123ns  +90.00%".
var (
	deltaRegex   = regexp.MustCompile(`([+-]\d+\.?\d*)%`)
	sectionRegex = regexp.MustCompile(`│\s+([^|\s]+)\s+vs base\s+│`)
)

func Run(scanner *bufio.Scanner, threshold float64) bool {
	regMap, pkgOrder, sectionOrder, osTxt, archTxt, cpuTxt := parseData(scanner, threshold)

	return processResults(regMap, pkgOrder, sectionOrder, threshold, osTxt, archTxt, cpuTxt)
}

func parseData(scanner *bufio.Scanner, threshold float64) (
	map[string]map[string][]string,
	[]string,
	[]string,
	string,
	string,
	string,
) {
	regMap := map[string]map[string][]string{} // Regressions list by packages
	currentPkg := unknownPackage               // Default value in case of no package header !
	currentSection := unknownSection

	var (
		osTxt        string
		archTxt      string
		cpuTxt       string
		pkgOrder     []string
		sectionOrder []string
	)

	sectionMap := map[string]struct{}{}

	for scanner.Scan() {
		line := scanner.Text()
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
	case *osTxt == "" && strings.HasPrefix(line, osPrefix):
		*osTxt = strings.TrimSpace(strings.TrimPrefix(line, osPrefix))

		return true
	case *archTxt == "" && strings.HasPrefix(line, archPrefix):
		*archTxt = strings.TrimSpace(strings.TrimPrefix(line, archPrefix))

		return true
	case *cpuTxt == "" && strings.HasPrefix(line, cpuPrefix):
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
			return fmt.Sprintf("%s (%.2f%% slower)", strings.Fields(line)[0], delta)
		}
	}

	return ""
}

func processResults(
	regMap map[string]map[string][]string,
	pkgOrder []string,
	sectionOrder []string,
	threshold float64,
	osTxt string,
	archTxt string,
	cpuTxt string,
) bool {
	if len(regMap) > 0 {
		slog.Error(fmt.Sprintf("Performance regression detected (threshold: %.1f%%):\n", threshold))
		slog.Error(fmt.Sprintf("Os %q / Arch %q / CPU %q", osTxt, archTxt, cpuTxt))

		// Sort packages and section in order to have a deterministic output (way easier for tests)
		inOrderMapIteratorHelper(regMap, pkgOrder, func(pkg string, subBegMap map[string][]string) {
			slog.Error("Package: " + pkg)
			inOrderMapIteratorHelper(subBegMap, sectionOrder, func(section string, regList []string) {
				slog.Error("  " + section)

				for _, reg := range regList {
					slog.Error("    " + reg)
				}
			})
		})

		return false
	}

	slog.Info("All good 🎉.")

	return true
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
