package main

import (
	"testing"
	"time"
)

func TestSimulationClockRetainsFractionalTime(t *testing.T) {
	var clock simulationClock
	var elapsed time.Duration
	total := 0
	for _, gap := range []time.Duration{8 * time.Millisecond, 8 * time.Millisecond, 17 * time.Millisecond,
		33 * time.Millisecond, 16 * time.Millisecond, 49 * time.Millisecond, 50 * time.Millisecond} {
		elapsed += gap
		steps, stalled := clock.advance(gap)
		total += steps
		if stalled || total != int(elapsed/simulationStep) {
			t.Fatalf("after %v: %d steps, want %d (stalled=%v)", elapsed, total, elapsed/simulationStep, stalled)
		}
	}
}

func TestSimulationClockBoundsRecoveryAndResetsAfterStall(t *testing.T) {
	var clock simulationClock
	if steps, stalled := clock.advance(10*simulationStep + simulationStep/2); steps != maxCatchUp || stalled {
		t.Fatalf("slow frame: %d steps, stalled=%v", steps, stalled)
	}
	if steps, _ := clock.advance(simulationStep - simulationStep/2); steps != 1 {
		t.Fatalf("lost fractional time or retained excess backlog: %d steps", steps)
	}
	clock.advance(simulationStep / 2)
	if steps, stalled := clock.advance(stallLimit + 1); steps != 0 || !stalled {
		t.Fatalf("stall: %d steps, stalled=%v", steps, stalled)
	}
	if steps, stalled := clock.advance(simulationStep / 2); steps != 0 || stalled {
		t.Fatalf("stall left old simulation time: %d steps, stalled=%v", steps, stalled)
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
	if got := mode.refreshPeriod(); got != simulationStep {
		t.Fatalf("pixel clock and blanking produced %v, want %v", got, simulationStep)
	}
	for _, invalid := range []fbVariable{{}, {Width: 480, Height: 800, PixelClock: 1},
		{Width: 480, Height: 800, PixelClock: 33334, VMode: 1},
		{Width: 480, Height: 800, PixelClock: 33334, Left: ^uint32(0)}} {
		if got := invalid.refreshPeriod(); got != time.Second/60 {
			t.Errorf("invalid timing %v produced %v", invalid, got)
		}
	}
}
