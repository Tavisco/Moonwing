package main

import "time"

const stallLimit = 250 * time.Millisecond

// Advance once per presented frame. Fractional-step accumulation at 60 Hz
// repeats frames on panels such as R1's 61.33 Hz display, causing judder.
// Long stalls still pause a run instead of advancing it unseen.
func shouldAdvanceFrame(elapsed time.Duration) bool {
	return elapsed <= stallLimit
}

type framePacer struct {
	period time.Duration
	next   time.Time
}

func (p *framePacer) finish(now time.Time, flipWait time.Duration) {
	if flipWait >= p.period/4 {
		// A substantial wait inside pan means the driver is pacing us.
		// Start the next frame immediately, even if reported mode timings
		// differ slightly from the panel's actual refresh interval.
		p.next = now
		return
	}
	p.next = p.next.Add(p.period)
	if p.next.Before(now) {
		// A blocking page flip may already have waited for the next refresh.
		// Do not add another frame's sleep or replay missed deadlines.
		p.next = now
	}
}

func (v fbVariable) refreshPeriod() time.Duration {
	const fallback = time.Second / 60
	horizontal := uint64(v.Width) + uint64(v.Left) + uint64(v.Right) + uint64(v.HSync)
	vertical := uint64(v.Height) + uint64(v.Upper) + uint64(v.Lower) + uint64(v.VSync)
	if v.PixelClock == 0 || v.VMode&3 != 0 || horizontal == 0 || vertical == 0 ||
		horizontal > 65536 || vertical > 65536 || v.PixelClock > 1000000 {
		return fallback
	}
	// fbdev's pixel clock is in picoseconds; time.Duration uses nanoseconds.
	period := time.Duration(horizontal * vertical * uint64(v.PixelClock) / 1000)
	if period < time.Second/120 || period > time.Second/20 {
		return fallback
	}
	return period
}
