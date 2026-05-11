package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"

	benchreg "github.com/yoanm/go-bench-regression"
)

func main() {
	os.Exit(execute())
}

const (
	missingArgsExitCode        = 1
	invalidThresholdExitCode   = 2
	missingInputExitCode       = 3
	regressionDetectedExitCode = 4
)

func execute() int {
	var threshold float64 // Acceptable regression (10% for instance)

	var err error

	cmdName := path.Base(os.Args[0])

	if len(os.Args) != 2 { //nolint:mnd // Useless to move it as constant
		slog.Error(fmt.Sprintf("Missing threshold argument. Usage: %s [threshold_percentage]\n", cmdName))

		return missingArgsExitCode
	} else if threshold, err = strconv.ParseFloat(os.Args[1], 64); err != nil {
		slog.Error("Threshold must be a valid float")

		return invalidThresholdExitCode
	} else if threshold >= 100 || threshold <= 0 {
		slog.Error("Threshold must be greater than 0% and lower than 100%")

		return invalidThresholdExitCode
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 { // Data must come from pipe !
		slog.Error(
			"No input detected. Please pipe benchstat output into this tool: " +
				"cat benchstat.out | " + cmdName + " [threshold_percentage]",
		)

		return missingInputExitCode
	}

	if !benchreg.Run(bufio.NewScanner(os.Stdin), threshold) {
		return regressionDetectedExitCode
	}

	return 0
}
