// Package metrics provides request rate tracking and analysis.
package metrics

import (
	"sync"
	"time"
)

// RateTracker tracks request rates over time using a circular buffer.
type RateTracker struct {
	mu           sync.RWMutex
	buckets      []int       // Requests per time bucket
	timestamps   []time.Time // Timestamp for each bucket
	currentIndex int         // Current position in circular buffer
	bucketSize   time.Duration
	windowSize   int // Number of buckets to keep
	totalCount   int // Total requests across all buckets
}

// NewRateTracker creates a new RateTracker.
// bucketSize: duration of each time bucket (e.g., 10 seconds)
// windowSize: number of buckets to keep (e.g., 60 = 10 minutes with 10s buckets)
func NewRateTracker(bucketSize time.Duration, windowSize int) *RateTracker {
	return &RateTracker{
		buckets:      make([]int, windowSize),
		timestamps:   make([]time.Time, windowSize),
		bucketSize:   bucketSize,
		windowSize:   windowSize,
		currentIndex: 0,
	}
}

// Record records a single request at the given time.
func (rt *RateTracker) Record(t time.Time) {
	rt.RecordN(t, 1)
}

// RecordN records n requests at the given time.
func (rt *RateTracker) RecordN(t time.Time, n int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Find or create bucket for this timestamp
	bucketTime := t.Truncate(rt.bucketSize)

	// Check if we need to advance to a new bucket
	if rt.timestamps[rt.currentIndex].IsZero() || bucketTime.After(rt.timestamps[rt.currentIndex]) {
		// Move to next bucket if time has advanced
		if !rt.timestamps[rt.currentIndex].IsZero() && bucketTime.Sub(rt.timestamps[rt.currentIndex]) >= rt.bucketSize {
			// Advance to new bucket
			rt.currentIndex = (rt.currentIndex + 1) % rt.windowSize

			// Clear old bucket data
			rt.totalCount -= rt.buckets[rt.currentIndex]
			rt.buckets[rt.currentIndex] = 0
			rt.timestamps[rt.currentIndex] = bucketTime
		}
	}

	// Add to current bucket
	rt.buckets[rt.currentIndex] += n
	rt.totalCount += n
	rt.timestamps[rt.currentIndex] = bucketTime
}

// Stats represents rate statistics.
type Stats struct {
	Current     float64 // Current requests per second
	Peak        float64 // Peak requests per second in window
	Average     float64 // Average requests per second
	Total       int     // Total requests in window
	TrendChange float64 // Percentage change vs previous period (positive = increasing)
}

// GetStats returns current rate statistics.
func (rt *RateTracker) GetStats() Stats {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if rt.totalCount == 0 {
		return Stats{}
	}

	now := time.Now()
	var validBuckets int
	var peak int
	var recentTotal int
	var previousTotal int

	// Calculate stats from valid buckets
	for i := 0; i < rt.windowSize; i++ {
		if rt.timestamps[i].IsZero() {
			continue
		}

		// Skip buckets older than our window
		age := now.Sub(rt.timestamps[i])
		if age > rt.bucketSize*time.Duration(rt.windowSize) {
			continue
		}

		validBuckets++

		// Track peak
		if rt.buckets[i] > peak {
			peak = rt.buckets[i]
		}

		// Recent vs previous for trend
		// Recent = buckets from last half of window
		// Previous = buckets from first half of window
		halfWindowDuration := rt.bucketSize * time.Duration(rt.windowSize/2)
		if age <= halfWindowDuration {
			recentTotal += rt.buckets[i]
		} else {
			previousTotal += rt.buckets[i]
		}
	}

	if validBuckets == 0 {
		return Stats{}
	}

	// Calculate rates (requests per second)
	bucketSeconds := rt.bucketSize.Seconds()
	peakRate := float64(peak) / bucketSeconds
	avgRate := float64(rt.totalCount) / (float64(validBuckets) * bucketSeconds)

	// Current rate from most recent bucket
	currentRate := float64(rt.buckets[rt.currentIndex]) / bucketSeconds

	// Calculate trend
	var trendChange float64
	if previousTotal > 0 {
		trendChange = (float64(recentTotal) - float64(previousTotal)) / float64(previousTotal) * 100
	}

	return Stats{
		Current:     currentRate,
		Peak:        peakRate,
		Average:     avgRate,
		Total:       rt.totalCount,
		TrendChange: trendChange,
	}
}

// Reset clears all tracking data.
func (rt *RateTracker) Reset() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for i := range rt.buckets {
		rt.buckets[i] = 0
		rt.timestamps[i] = time.Time{}
	}
	rt.currentIndex = 0
	rt.totalCount = 0
}
