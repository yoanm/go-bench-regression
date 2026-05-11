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

func execute() int {
	var threshold float64 // Acceptable regression (10% for instance)

	var err error

	cmdName := path.Base(os.Args[0])

	if len(os.Args) != 2 { //nolint:mnd // Useless to move it as constant
		slog.Error(fmt.Sprintf("Missing threshold argument. Usage: %s [threshold_percentage]\n", cmdName))

		return 1
	} else if threshold, err = strconv.ParseFloat(os.Args[1], 64); err != nil {
		slog.Error("Threshold must be a valid float")

		return 2 //nolint:mnd // Useless to move it as constant
	} else if threshold >= 100 || threshold <= 0 {
		slog.Error("Threshold must be greater than 0% and lower than 100%")

		return 2 //nolint:mnd // Useless to move it as constant
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 { // Data must come from pipe !
		slog.Error(
			"No input detected. Please pipe benchstat output into this tool: " +
				"cat benchstat.out | " + cmdName + " [threshold_percentage]",
		)

		return 3 //nolint:mnd // Useless to move it as constant
	}

	if !benchreg.Run(bufio.NewScanner(os.Stdin), threshold) {
		return 4 //nolint:mnd // Useless to move it as constant
	}

	return 0
}
