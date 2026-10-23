package migrate

import (
	"math"
	"testing"
)

func TestDollarsToCents(t *testing.T) {
	cases := []struct {
		in   float64
		want int64
		null bool
	}{
		{0, 0, true}, // legacy "contact for price"
		{-5, 0, true},
		{math.NaN(), 0, true},
		{12.5, 1250, false},
		{19.99, 1999, false}, // float rounding must not produce 1998
		{0.015, 2, false},
	}
	for _, c := range cases {
		got := dollarsToCents(c.in)
		if c.null != (got == nil) || (got != nil && *got != c.want) {
			t.Errorf("dollarsToCents(%v) = %v, want null=%v %d", c.in, got, c.null, c.want)
		}
	}
}
