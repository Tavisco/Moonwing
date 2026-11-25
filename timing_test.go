package main

import (
	"testing"
	"time"
)

func TestEveryR1RefreshAdvancesSimulation(t *testing.T) {
	const period = 16304 * time.Microsecond
	for frame := 0; frame < 1000; frame++ {
		if !shouldAdvanceFrame(period) {
			t.Fatalf("frame %d repeated at R1's refresh interval", frame)
		}
	}
}

func TestFrameAdvanceDoesNotAccumulateOrCatchUp(t *testing.T) {
	for _, gap := range []time.Duration{0, 15 * time.Millisecond, 16304 * time.Microsecond,
		18 * time.Millisecond, 33 * time.Millisecond, 65 * time.Millisecond, stallLimit} {
		if !shouldAdvanceFrame(gap) {
			t.Fatalf("frame gap %v did not receive its single update", gap)
		}
	}
	if shouldAdvanceFrame(stallLimit + time.Nanosecond) {
		t.Fatal("a long stall should pause the active run")
	}
	if !shouldAdvanceFrame(16304 * time.Microsecond) {
		t.Fatal("the frame after a long stall did not resume with one update")
	}
}

func TestFramePacerAccountsForBlockingPresentation(t *testing.T) {
	start := time.Unix(1, 0)
	period := time.Second / 60
	p := framePacer{period: period, next: start}
	p.finish(start.Add(2*time.Millisecond), 0)
	if p.next != start.Add(period) {
		t.Fatalf("fast frame deadline = %v", p.next)
	}
	finished := start.Add(3 * period)
	p.finish(finished, 0)
	if p.next != finished {
		t.Fatalf("late presentation added a wait: deadline %v, finished %v", p.next, finished)
	}
	p.finish(finished.Add(2*time.Millisecond), 0)
	if p.next != finished.Add(period) {
		t.Fatalf("pacer did not recover after missed frames: %v", p.next)
	}
	finished = finished.Add(period + 15*time.Millisecond)
	p.finish(finished, 13*time.Millisecond)
	if p.next != finished {
		t.Fatalf("blocking pan added a second pacing delay: %v", p.next)
	}
}

func TestFramebufferRefreshPeriod(t *testing.T) {
	mode := fbVariable{Width: 480, Height: 800, Left: 10, Right: 5, HSync: 5,
		Upper: 100, Lower: 50, VSync: 50, PixelClock: 33334}
	if got := mode.refreshPeriod(); got != 16667*time.Microsecond {
		t.Fatalf("pixel clock and blanking produced %v, want %v", got, 16667*time.Microsecond)
	}
	for _, invalid := range []fbVariable{{}, {Width: 480, Height: 800, PixelClock: 1},
		{Width: 480, Height: 800, PixelClock: 33334, VMode: 1},
		{Width: 480, Height: 800, PixelClock: 33334, Left: ^uint32(0)}} {
		if got := invalid.refreshPeriod(); got != time.Second/60 {
			t.Errorf("invalid timing %v produced %v", invalid, got)
		}
	}
}
