package main

import "testing"

func TestExitAndPlayHitTargets(t *testing.T) {
	const h = 400
	g := newGame(h, 0, 1)
	g.fadeTick = -1
	b := g.startBounds()
	for _, tc := range []struct {
		pos  point
		want action
	}{
		{point{212, 27}, exitAction},
		{point{27, 27}, noAction},
		{point{212, h - 28}, noAction},
		{point{16, 354}, noAction},
		{point{120, b.y + 10}, flapAction},
		{point{120, 25}, noAction},
	} {
		if actual := g.touchAction(tc.pos, h); actual != tc.want {
			t.Errorf("tap at %v yielded %v, want %v", tc.pos, actual, tc.want)
		}
	}
}

func TestR1PanelCoordinatesMatchVisibleButtons(t *testing.T) {
	// hyn_ts advertises a 720x1280 sensor but emits 480x800 panel pixels.
	// Rescaling against that sensor range used to miss the right-hand exit;
	// the old multi-corner workaround then made a left-hand tap close instead.
	const height = 400
	sensor := touchAxes{absInfo{Max: 720}, absInfo{Max: 1280}}
	axes := sensor.calibrated("r1", "hyn_ts", 480, 800)
	g := newGame(height, 0, 1)
	g.fadeTick = -1
	for _, state := range []phase{title, ready, flying, paused, falling, results} {
		g.state, g.overTick = state, overDoneAt
		for _, tc := range []struct {
			raw  contactPoint
			want action
		}{
			{contactPoint{x: 424, y: 54}, exitAction},
			{contactPoint{x: 54, y: 54}, noAction},
		} {
			want := tc.want
			if state != title && state != results && want == noAction {
				want = flapAction
			}
			p := axes.screenPoint(tc.raw, 480, 800, height)
			if got := g.touchAction(p, height); got != want {
				t.Errorf("state %d, panel tap %v mapped to %v: got %v, want %v", state, tc.raw, p, got, want)
			}
		}
		if g.canStart() {
			b := g.startBounds()
			raw := contactPoint{x: int32((b.x + b.w/2) * 2), y: int32((b.y + b.h/2) * 2)}
			if got := g.touchAction(axes.screenPoint(raw, 480, 800, height), height); got != flapAction {
				t.Errorf("visible Fly button missed in state %d", state)
			}
		}
	}
	// The exception must not change devices whose advertised ranges are real.
	for _, device := range [][2]string{{"r3proii", "goodix-ts"}, {"r1", "other"}, {"r3ii_2025", "hyn_ts"}} {
		if got := sensor.calibrated(device[0], device[1], 480, 800); got != sensor {
			t.Errorf("changed calibration for %v", device)
		}
	}
}

func TestSensorRangeDoesNotGuessRotation(t *testing.T) {
	axes := touchAxes{absInfo{Min: 100, Max: 4195}, absInfo{Min: 200, Max: 2247}}
	// Different X/Y sensor resolutions do not mean the panel is rotated.
	p := axes.screenPoint(contactPoint{x: 4195, y: 200}, 480, 800, 400)
	if p != (point{239, 0}) {
		t.Fatalf("top right mapped to %v", p)
	}
}

func TestTouchUsesActiveSlot(t *testing.T) {
	const height = 400
	axes := touchAxes{absInfo{Max: 479}, absInfo{Max: 799}}
	in := touchState{
		multi: axes, hasSlots: true,
		pos: contactPoint{x: 240, y: 400, hasX: true, hasY: true},
	}
	in.event(evAbs, absSlot, 2)
	in.event(evAbs, absTracking, 8)
	in.event(evAbs, absMTX, 424)
	in.event(evAbs, absMTY, 54)
	in.event(evKey, btnTouch, 1)
	p, a, pressed := in.event(evSync, synReport, 0)
	if !pressed {
		t.Fatal("tap in slot 2 was missed")
	}
	g := newGame(height, 0, 1)
	got := g.touchAction(a.screenPoint(p, 480, 800, height), height)
	if got != exitAction {
		t.Fatalf("tap on × yielded %v, want exit", got)
	}
	if _, _, pressed := in.event(evSync, synReport, 0); pressed {
		t.Fatal("a held finger fired a second tap")
	}
}

func TestUnpositionedTouchDoesNotFire(t *testing.T) {
	var in touchState
	in.event(evKey, btnTouch, 1)
	if _, _, pressed := in.event(evSync, synReport, 0); pressed {
		t.Fatal("button press without touch coordinates triggered an action")
	}
	in.event(evAbs, absX, 424)
	in.event(evAbs, absY, 54)
	p, a, pressed := in.event(evSync, synReport, 0)
	if !pressed {
		t.Fatal("positioned touch failed to fire")
	}
	const h = 400
	g := newGame(h, 0, 1)
	if result := g.touchAction(a.screenPoint(p, 480, 800, h), h); result != exitAction {
		t.Fatalf("touch with missing axis ranges yielded %v, want exit", result)
	}
}
