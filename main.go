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
	pkgPrefix  = "pkg: "
	osPrefix   = "goos: "
	archPrefix = "goarch: "
	cpuPrefix  = "cpu: "
)

// Match lines like: "BenchmarkAbc-42  230ns  123ns  +90.00%".
var deltaRegex = regexp.MustCompile(`([+-]\d+\.?\d*)%`) // TODO: Add "Benchmark" header to the regexp ??

func Run(scanner *bufio.Scanner, threshold float64) bool {
	regMap, osTxt, archTxt, cpuTxt := parseData(scanner, threshold)

	return processResults(regMap, threshold, osTxt, archTxt, cpuTxt)
}

func parseData(scanner *bufio.Scanner, threshold float64) (map[string][]string, string, string, string) {
	regMap := map[string][]string{} // Regressions list by packages
	currentPkg := "UNKNOWN"         // Default value in case of no package header !

	var (
		osTxt   string
		archTxt string
		cpuTxt  string
	)

	for scanner.Scan() {
		line := scanner.Text()
		// Check if lines are related to package, os, arch or cpu info
		switch {
		case strings.HasPrefix(line, pkgPrefix):
			currentPkg = strings.TrimSpace(strings.TrimPrefix(line, pkgPrefix))

			continue // No need to parse the line further, we have what we want
		case osTxt == "" && strings.HasPrefix(line, osPrefix):
			osTxt = strings.TrimSpace(strings.TrimPrefix(line, osPrefix))

			continue // No need to parse the line further, we have what we want
		case archTxt == "" && strings.HasPrefix(line, archPrefix):
			archTxt = strings.TrimSpace(strings.TrimPrefix(line, archPrefix))

			continue // No need to parse the line further, we have what we want
		case cpuTxt == "" && strings.HasPrefix(line, cpuPrefix):
			cpuTxt = strings.TrimSpace(strings.TrimPrefix(line, cpuPrefix))

			continue // No need to parse the line further, we have what we want
		}

		if txt := parseLine(line, threshold); txt != "" {
			regMap[currentPkg] = append(regMap[currentPkg], txt)
		}
	}

	return regMap, osTxt, archTxt, cpuTxt
}

func parseLine(line string, threshold float64) string {
	if matches := deltaRegex.FindStringSubmatch(line); len(matches) > 1 {
		delta, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			slog.Error("Error parsing delta. Skipping line", "line", line)

			return ""
		}

		// Positive delta means regression (slower)
		if delta > threshold {
			return fmt.Sprintf("  %s (%.2f%% slower)", strings.Fields(line)[0], delta)
		}
	}

	return ""
}

func processResults(regMap map[string][]string, threshold float64, osTxt string, archTxt string, cpuTxt string) bool {
	if len(regMap) > 0 {
		slog.Error(fmt.Sprintf("Performance regression detected (threshold: %.1f%%):\n", threshold))
		slog.Error(fmt.Sprintf("Os %q / Arch %q / CPU %q", osTxt, archTxt, cpuTxt))

		for pkg, regList := range regMap {
			slog.Error("Package: " + pkg)

			for _, reg := range regList {
				slog.Error(reg)
			}
		}

		return false
	}

	slog.Info("All good 🎉.")

	return true
}
