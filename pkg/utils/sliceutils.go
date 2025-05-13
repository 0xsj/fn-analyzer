// pkg/utils/sliceutils.go
package utils

import (
	"math"
	"sort"
	"time"
)

func CalculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	idx := int(math.Floor(float64(len(durations)) * percentile / 100.0))

	if idx >= len(durations) {
		idx = len(durations) - 1
	}

	return durations[idx]
}

func CalculateStandardDeviation(durations []time.Duration, mean time.Duration) time.Duration {
	if len(durations) <= 1 {
		return 0
	}

	var sum int64
	for _, d := range durations {
		diff := d - mean
		sum += diff.Nanoseconds() * diff.Nanoseconds()
	}

	variance := float64(sum) / float64(len(durations)-1)
	return time.Duration(math.Sqrt(variance))
}

type Stats struct {
	Min     time.Duration
	Max     time.Duration
	Mean    time.Duration
	Median  time.Duration
	StdDev  time.Duration
	P95     time.Duration
	P99     time.Duration
	Samples int
}

func CalculateStats(durations []time.Duration) Stats {
	if len(durations) == 0 {
		return Stats{}
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	mean := total / time.Duration(len(durations))

	var sumSquares int64
	for _, d := range durations {
		diff := d - mean
		sumSquares += diff.Nanoseconds() * diff.Nanoseconds()
	}

	variance := float64(sumSquares) / float64(len(durations))
	stdDev := time.Duration(math.Sqrt(variance))

	p50Idx := int(float64(len(durations)) * 0.5)
	p95Idx := int(float64(len(durations)) * 0.95)
	p99Idx := int(float64(len(durations)) * 0.99)

	if p50Idx >= len(durations) {
		p50Idx = len(durations) - 1
	}
	if p95Idx >= len(durations) {
		p95Idx = len(durations) - 1
	}
	if p99Idx >= len(durations) {
		p99Idx = len(durations) - 1
	}

	return Stats{
		Min:     durations[0],
		Max:     durations[len(durations)-1],
		Mean:    mean,
		Median:  durations[p50Idx],
		StdDev:  stdDev,
		P95:     durations[p95Idx],
		P99:     durations[p99Idx],
		Samples: len(durations),
	}
}
