package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestRandomizedIntervalRange(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	base := 200 * time.Millisecond
	min := base - base/4
	max := base + base/4
	for i := 0; i < 1000; i++ {
		got := randomizedInterval(rnd, base)
		if got < min || got > max {
			t.Fatalf("interval out of range: got=%v min=%v max=%v", got, min, max)
		}
	}
}

func TestDataGeneratorGeneratesData(t *testing.T) {
	codes := []string{"AAA", "BBB", "CCC"}
	dg := NewDataGenerator(codes, 5*time.Millisecond)
	ch := make(chan Data, 100) // Increased buffer for batch generation
	go dg.Start(func(d Data) {
		ch <- d
	})

	deadline := time.After(300 * time.Millisecond)
	count := 0
	for count < 5 {
		select {
		case d := <-ch:
			if d.Timestamp.IsZero() {
				t.Fatalf("timestamp is zero")
			}
			if d.Code != "AAA" && d.Code != "BBB" && d.Code != "CCC" {
				t.Fatalf("unexpected code: %q", d.Code)
			}
			if d.Price <= 0 {
				t.Fatalf("price not positive: %v", d.Price)
			}
			if d.Volume <= 0 {
				t.Fatalf("volume not positive: %v", d.Volume)
			}
			if d.BidPrice > d.Price || d.AskPrice < d.Price || d.AskPrice <= d.BidPrice {
				t.Fatalf("invalid bid/ask: bid=%v price=%v ask=%v", d.BidPrice, d.Price, d.AskPrice)
			}
			count++
		case <-deadline:
			dg.Stop()
			t.Fatalf("timed out waiting for data")
		}
	}

	dg.Stop()
}

func TestDataGeneratorBatchGeneration(t *testing.T) {
	// Test that short intervals generate multiple data points per tick
	codes := []string{"AAA"}
	dg := NewDataGenerator(codes, 1*time.Millisecond) // Very short interval
	ch := make(chan Data, 500)

	go dg.Start(func(d Data) {
		ch <- d
	})

	time.Sleep(250 * time.Millisecond) // Wait for batch generation
	dg.Stop()

	count := len(ch)
	// With 1ms interval and 100ms ticks, should get ~100 per tick
	// In 250ms, expect around 200-250 data points
	if count < 150 {
		t.Fatalf("expected at least 150 data points for batch generation, got %d", count)
	}
	t.Logf("generated %d data points in 250ms (interval=1ms)", count)
}

func TestDataGeneratorStopIdempotent(t *testing.T) {
	codes := []string{"AAA"}
	dg := NewDataGenerator(codes, 5*time.Millisecond)
	done := make(chan struct{})
	go func() {
		dg.Start(func(Data) {})
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	dg.Stop()
	dg.Stop()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("generator did not stop")
	}
}
