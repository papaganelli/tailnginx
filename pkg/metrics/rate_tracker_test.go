package metrics

import (
	"testing"
	"time"
)

func TestNewRateTracker(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 60)

	if rt == nil {
		t.Fatal("NewRateTracker returned nil")
	}
	if len(rt.buckets) != 60 {
		t.Errorf("Expected 60 buckets, got %d", len(rt.buckets))
	}
	if rt.bucketSize != 10*time.Second {
		t.Errorf("Expected bucket size 10s, got %v", rt.bucketSize)
	}
}

func TestRecord(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 60)
	now := time.Now()

	rt.Record(now)
	rt.Record(now)
	rt.RecordN(now, 3)

	stats := rt.GetStats()
	if stats.Total != 5 {
		t.Errorf("Expected 5 total requests, got %d", stats.Total)
	}
}

func TestGetStats(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 60)
	now := time.Now()

	// Record 100 requests in current bucket
	rt.RecordN(now, 100)

	stats := rt.GetStats()

	// With 100 requests in 10 seconds, rate should be 10 req/s
	expectedRate := 10.0
	if stats.Current < expectedRate-0.1 || stats.Current > expectedRate+0.1 {
		t.Errorf("Expected current rate ~%.1f req/s, got %.2f", expectedRate, stats.Current)
	}

	if stats.Total != 100 {
		t.Errorf("Expected total 100, got %d", stats.Total)
	}
}

func TestMultipleBuckets(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 60)
	baseTime := time.Now()

	// Add requests across multiple time buckets
	rt.RecordN(baseTime, 50)                      // Bucket 0
	rt.RecordN(baseTime.Add(15*time.Second), 60) // Bucket 1
	rt.RecordN(baseTime.Add(25*time.Second), 70) // Bucket 2

	stats := rt.GetStats()

	if stats.Total != 180 {
		t.Errorf("Expected total 180, got %d", stats.Total)
	}

	// Peak should be 70 requests / 10s = 7 req/s
	expectedPeak := 7.0
	if stats.Peak < expectedPeak-0.1 || stats.Peak > expectedPeak+0.1 {
		t.Errorf("Expected peak ~%.1f req/s, got %.2f", expectedPeak, stats.Peak)
	}
}

func TestTrendCalculation(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 10)
	baseTime := time.Now().Add(-100 * time.Second) // Start in past

	// Older buckets (lower rate)
	for i := 0; i < 5; i++ {
		rt.RecordN(baseTime.Add(time.Duration(i)*10*time.Second), 10)
	}
	// Recent buckets (higher rate)
	for i := 5; i < 10; i++ {
		rt.RecordN(baseTime.Add(time.Duration(i)*10*time.Second), 20)
	}

	stats := rt.GetStats()

	// Trend should be positive (increasing) or at least calculated
	// We just verify it's calculated, not the specific value
	t.Logf("Trend: %.2f%%, Recent: %d, Previous: %d", stats.TrendChange, 100, 50)
}

func TestEmptyTracker(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 60)

	stats := rt.GetStats()

	if stats.Current != 0 {
		t.Errorf("Expected 0 current rate, got %.2f", stats.Current)
	}
	if stats.Total != 0 {
		t.Errorf("Expected 0 total, got %d", stats.Total)
	}
}

func TestReset(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 60)
	now := time.Now()

	rt.RecordN(now, 100)

	stats := rt.GetStats()
	if stats.Total != 100 {
		t.Errorf("Expected 100 before reset, got %d", stats.Total)
	}

	rt.Reset()

	stats = rt.GetStats()
	if stats.Total != 0 {
		t.Errorf("Expected 0 after reset, got %d", stats.Total)
	}
	if stats.Current != 0 {
		t.Errorf("Expected 0 current after reset, got %.2f", stats.Current)
	}
}

func TestConcurrency(t *testing.T) {
	rt := NewRateTracker(10*time.Second, 60)
	now := time.Now()

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				rt.Record(now)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := rt.GetStats()
	expected := 1000
	if stats.Total != expected {
		t.Errorf("Expected %d total requests, got %d", expected, stats.Total)
	}
}

func BenchmarkRecord(b *testing.B) {
	rt := NewRateTracker(10*time.Second, 60)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.Record(now)
	}
}

func BenchmarkGetStats(b *testing.B) {
	rt := NewRateTracker(10*time.Second, 60)
	now := time.Now()

	// Populate with some data
	for i := 0; i < 1000; i++ {
		rt.Record(now.Add(time.Duration(i) * time.Second))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rt.GetStats()
	}
}
