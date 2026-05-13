# benchreg

Package benchreg analyzes benchstat output and detects performance regressions
above a specified threshold. It parses benchstat command output, extracts regression
percentages from benchmark results, and reports those exceeding the threshold.

Usage:

```go
scanner := bufio.NewScanner(os.Stdin)
success := benchreg.Run(scanner, 10.0) // 10% threshold
if !success {
	fmt.Println("Performance regressions detected")
}
```

## Functions

### func [Run](/main.go#L69)

`func Run(input *bufio.Scanner, threshold float64, output io.StringWriter) bool`

Run analyzes benchstat output from the provided scanner and detects performance
regressions exceeding the specified threshold percentage.

It parses benchstat output to extract:

```diff
- Package names and benchmark results
- Regression percentages (positive deltas)
- Metadata (operating system, architecture, CPU)
```

Parameters:

```diff
- scanner: reads benchstat output line by line
- threshold: percentage threshold; regressions above this value are reported
- output: writer for the summary
```

Returns:

```diff
- true if no regressions are detected (all deltas at or below threshold)
- false if regressions are detected and logged
```

Example output format:

```go
pkg: mypackage
goos: linux
goarch: amd64
cpu: Intel(R) Core(TM)
BenchmarkFunc-8    100  100ns  120ns  +20.00%
```

With threshold 10%, the above example would trigger a regression report for the 20% delta.

## Sub Packages

* [cmd/bench-reg](./cmd/bench-reg)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
