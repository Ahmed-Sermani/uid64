package uid64_test

import (
	"testing"

	"github.com/Ahmed-Sermani/uid64"
)

var g = uid64.New()

func Benchmark(b *testing.B) {
	for n := 0; n < b.N; n++ {
		g.NextID()
	}
}
