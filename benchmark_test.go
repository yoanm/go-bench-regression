package benchreg_test

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	benchreg "github.com/yoanm/go-bench-regression"
)

func BenchmarkRun(b *testing.B) {
	input := `pkg: testpkg
goos: linux
goarch: amd64
cpu: Intel(R) Core(TM) i7-8700K
BenchmarkFunc1-8     1000  100ns  110ns  +10.00%
BenchmarkFunc2-8     1000  200ns  210ns  +5.00%
BenchmarkFunc3-8     1000  300ns  290ns  -3.33%`

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		scanner := bufio.NewScanner(strings.NewReader(input))
		output := &bytes.Buffer{}
		benchreg.Run(scanner, 10.0, output)
	}
}
