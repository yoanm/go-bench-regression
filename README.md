# go-bench-regression

A Go CLI tool for comparing benchstat output and detecting performance regressions above a specified threshold.

## Overview

`go-bench-regression` analyzes Go benchmark results (from `benchstat` command) and identifies performance regressions that exceed a configurable threshold. It's useful in CI/CD pipelines to automatically detect performance degradation in your Go projects.

## Features

- 📊 Parses `benchstat` output format
- 🎯 Configurable regression threshold (percentage-based)
- 📦 Supports multiple packages and benchmark sections
- 🔍 Detailed regression reporting with OS/Architecture/CPU metadata
- ⚡ Fast and efficient processing
- ✅ Proper exit codes for CI integration

## Installation

### Using `go install`

```bash
go install github.com/yoanm/go-bench-regression/cmd/bench-reg@latest
```

This installs the `bench-reg` CLI tool into your `$GOPATH/bin` directory. Make sure `$GOPATH/bin` is in your `$PATH`.

### From Source

```bash
git clone https://github.com/yoanm/go-bench-regression.git
cd go-bench-regression
make build
```

The binary will be available as `./bench-reg`.

## Usage

### Basic Usage

```bash
benchstat before.txt after.txt | bench-reg 10
```

This pipes benchstat output into `bench-reg` with a 10% threshold. Any regressions exceeding 10% will be reported.

### With File Input

```bash
cat benchstat_output.txt | bench-reg 5
```

### As a Library

```go
package main

import (
"bufio"
"os"

benchreg "github.com/yoanm/go-bench-regression"
)

func main() {
    scanner := bufio.NewScanner(os.Stdin) // or bufio.NewScanner(bytes.NewReader(inputBytes))
    
    if !benchreg.Run(scanner, 10.0) { // 10% threshold
        os.Exit(1) // Regressions detected
    }
}
```

## Exit Codes

The CLI tool returns specific exit codes for different scenarios:

| Exit Code | Meaning                                                        |
|----------:|:---------------------------------------------------------------|
|         0 | Success - No regressions detected                              |
|         1 | Regressions detected above threshold                           |
|         2 | Invalid arguments (missing or wrong number of arguments)       |
|         3 | Invalid threshold value (not a valid number, or out of range)  |
|         4 | No input detected (not piped from stdin)                       |

## Example

### Running benchmarks and detecting regressions

```bash
# Run benchmarks on your baseline
go test -bench=. ./... > baseline.txt

# Make changes to your code...

# Run benchmarks again
go test -bench=. ./... > after.txt

# Compare and detect regressions (threshold: 10%)
benchstat baseline.txt after.txt | bench-reg 10
```

### CI/CD Integration Example (GitHub Actions)

```yaml
- name: Run Benchmarks
  run: go test -run='^$' -bench=. -benchmem ./... > after.txt

- name: Compare with Baseline
  run: benchstat baseline.txt after.txt | bench-reg 10
```

## Threshold Guidelines

- **1-5%**: Strict threshold for performance-critical code
- **5-10%**: Typical choice for most projects
- **10-20%**: Lenient threshold for noisy benchmarks
- **20%+**: Very permissive threshold

## Features

### Input Format

The tool expects benchstat output format:

```
pkg: github.com/user/mypackage
goos: linux
goarch: amd64
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
                │ baseline.txt │          after.txt                  │
                │     B/op     │     B/op      vs base               │
BenchmarkFunc-4   2.269Mi ± 0%   2.270Mi ± 0%  +12.39% (p=0.128 n=7)

                │ baseline.txt │         after.txt                  │
                │  allocs/op   │  allocs/op   vs base               │
BenchmarkFunc-4    23.20k ± 0%   23.20k ± 0%  +11.75% (p=1.000 n=7)
```

The tool:
- ✅ Detects regressions (positive deltas exceeding threshold)
- ✅ Ignores improvements (negative deltas)
- ✅ Handles multiple packages and sections
- ✅ Extracts metadata (OS, Architecture, CPU)
- ✅ Supports geomean benchmarks

### Output Example

When regressions are detected:

```
ERROR Performance regression detected (threshold: 10.0%):
ERROR Os "linux" / Arch "amd64" / CPU "Intel(R) Core(TM)"
ERROR Package: github.com/user/mypackage
ERROR   B/op
ERROR     - BenchmarkFunc (12.39% slower)
ERROR   allocs/op
ERROR     - BenchmarkFunc (11.75% slower)
```

## Development

### Running Tests

```bash
# Install tools for dev environment
make configure-dev-env

# Install tools for test/CI environment
make configure-test-env

# Run all tests with coverage
make test

# Run tests only
make test-go

# Run linting
make test-lint

# Format code
make fmt
```

### Building Documentation

```bash
make build-doc
```

This regenerates `DOC.md` from Go function documentation using `goreadme`.

## License

See LICENSE file in the repository.
