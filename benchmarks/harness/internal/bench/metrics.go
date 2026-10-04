// Package bench provides small measurement primitives for load benchmarks:
// Histogram for latency distributions and Meter for throughput.
package bench

import (
	"math"
	"sort"
	"sync"
	"time"
)

// Histogram collects float64 samples. The unit is defined at the use site
// (gosamp-bench records milliseconds). All methods are safe for concurrent use.
type Histogram struct {
	mu   sync.Mutex
	vals []float64
}

// Add records one sample.
func (h *Histogram) Add(v float64) {
	h.mu.Lock()
	h.vals = append(h.vals, v)
	h.mu.Unlock()
}

// Count returns the number of samples.
func (h *Histogram) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.vals)
}

// snapshot returns a copy of the samples. Callers must not hold h.mu.
func (h *Histogram) snapshot() []float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]float64, len(h.vals))
	copy(out, h.vals)
	return out
}

// Mean returns the arithmetic mean, or NaN when empty.
func (h *Histogram) Mean() float64 {
	vals := h.snapshot()
	if len(vals) == 0 {
		return math.NaN()
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// Min returns the smallest sample, or NaN when empty.
func (h *Histogram) Min() float64 {
	vals := h.snapshot()
	if len(vals) == 0 {
		return math.NaN()
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// Max returns the largest sample, or NaN when empty.
func (h *Histogram) Max() float64 {
	vals := h.snapshot()
	if len(vals) == 0 {
		return math.NaN()
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// Quantile returns the q-th quantile with linear interpolation over sorted
// samples. q is clamped to [0,1]. It returns NaN when empty.
func (h *Histogram) Quantile(q float64) float64 {
	vals := h.snapshot()
	if len(vals) == 0 {
		return math.NaN()
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	sort.Float64s(vals)
	if len(vals) == 1 {
		return vals[0]
	}
	pos := q * float64(len(vals)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return vals[lo]
	}
	frac := pos - float64(lo)
	return vals[lo]*(1-frac) + vals[hi]*frac
}

// Meter measures throughput as events per second since the first mark.
// Use Start to reset with an explicit base time. All methods are safe for
// concurrent use.
type Meter struct {
	mu      sync.Mutex
	start   time.Time
	total   int64
	started bool
}

// Start resets the meter: the count returns to zero and the base time
// becomes t.
func (m *Meter) Start(t time.Time) {
	m.mu.Lock()
	m.start = t
	m.total = 0
	m.started = true
	m.mu.Unlock()
}

// Mark records n events. The first Mark without a prior Start uses
// time.Now() as the base time.
func (m *Meter) Mark(n int) {
	m.mu.Lock()
	if !m.started {
		m.start = time.Now()
		m.started = true
	}
	m.total += int64(n)
	m.mu.Unlock()
}

// Rate returns events per second over now.Sub(start). It returns 0 when no
// events were marked or the elapsed time is not positive.
func (m *Meter) Rate(now time.Time) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.total == 0 {
		return 0
	}
	el := now.Sub(m.start).Seconds()
	if el <= 0 {
		return 0
	}
	return float64(m.total) / el
}

// Count returns the total number of marked events.
func (m *Meter) Count() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.total
}
