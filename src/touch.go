package main

const (
	evSync      = 0
	evKey       = 1
	evAbs       = 3
	synReport   = 0
	synMTReport = 2
	synDropped  = 3
	absX        = 0
	absY        = 1
	absSlot     = 47
	absMTX      = 53
	absMTY      = 54
	absTracking = 57
	btnTouch    = 330
)

type absInfo struct {
	Value, Min, Max, Fuzz, Flat, Resolution int32
}

func (a absInfo) valid() bool { return a.Max > a.Min }

type point struct{ x, y int }

type touchAxes struct{ x, y absInfo }

type contactPoint struct {
	x, y       int32
	hasX, hasY bool
	updated    bool
}

func (p *contactPoint) position(horizontal bool, value int32) {
	if horizontal {
		p.x, p.hasX = value, true
	} else {
		p.y, p.hasY = value, true
	}
	p.updated = true
}

type touchSlot struct {
	pos              contactPoint
	active, reported bool
}

type touchState struct {
	single, multi                                         touchAxes
	pos                                                   contactPoint
	slots                                                 [16]touchSlot
	slot                                                  int
	slotMin                                               int32
	hasSlots, sawTouchKey, touching, down, dropped, extra bool
}

func (in *touchState) event(kind, code uint16, value int32) (contactPoint, touchAxes, bool) {
	if in.dropped {
		in.dropped = kind != evSync || code != synReport
		return contactPoint{}, touchAxes{}, false
	}
	switch kind {
	case evKey:
		if code == btnTouch {
			in.sawTouchKey, in.touching = true, value != 0
		}
	case evAbs:
		switch code {
		case absX, absY:
			in.pos.position(code == absX, value)
		case absSlot:
			in.hasSlots = true
			in.slot = int(value - in.slotMin)
		case absMTX, absMTY, absTracking:
			if in.slot < 0 || in.slot >= len(in.slots) || in.extra {
				break
			}
			slot := &in.slots[in.slot]
			slot.reported = true
			if code == absTracking {
				slot.active = value >= 0
			} else {
				slot.pos.position(code == absMTX, value)
				if !in.hasSlots {
					slot.active = true
				}
			}
		}
	case evSync:
		switch code {
		case synDropped:
			in.dropped, in.down, in.touching = true, false, false
			in.extra = false
			in.pos = contactPoint{}
			clear(in.slots[:])
		case synMTReport:
			in.extra = true
		case synReport:
			return in.report()
		}
	}
	return contactPoint{}, touchAxes{}, false
}

func (in *touchState) report() (contactPoint, touchAxes, bool) {
	isDown := in.touching
	pos, axes := in.pos, in.single
	// Both coordinate pairs can be advertised. Use coordinates from the
	// contact that actually reported, rather than an inactive MT slot.
	var active *touchSlot
	for i := range in.slots {
		slot := &in.slots[i]
		if slot.active && (in.hasSlots || slot.reported) &&
			(active == nil || (!active.pos.updated && slot.pos.updated)) {
			active = slot
		}
	}
	if active != nil {
		if !in.sawTouchKey {
			isDown = true
		}
		if active.pos.hasX && active.pos.hasY &&
			(!pos.hasX || !pos.hasY || !pos.updated && active.pos.updated) {
			pos, axes = active.pos, in.multi
		}
	}

	pressed := isDown && !in.down && pos.hasX && pos.hasY
	in.down = isDown && (in.down || pressed)
	in.pos.updated, in.extra = false, false
	for i := range in.slots {
		slot := &in.slots[i]
		slot.pos.updated, slot.reported = false, false
	}
	return pos, axes, pressed
}

func axisPixel(value int32, axis absInfo, extent, nativeExtent int) int {
	if !axis.valid() {
		axis.Min, axis.Max = 0, int32(nativeExtent-1)
	}
	span := int64(axis.Max) - int64(axis.Min)
	pos := int64(value) - int64(axis.Min)
	return int(min(max(pos, 0), span) * int64(extent-1) / span)
}

func (a touchAxes) calibrated(board, name string, width, height int) touchAxes {
	// Compás's R1 Hynitron driver already reports panel pixels, although
	// EVIOCGABS advertises the sensor's 720x1280 range. Match the firmware's
	// exception for this exact board/controller rather than guessing a rotation.
	if board == "r1" && name == "hyn_ts" {
		return touchAxes{absInfo{Max: int32(width - 1)}, absInfo{Max: int32(height - 1)}}
	}
	return a
}

func (a touchAxes) screenPoint(p contactPoint, nativeWidth, nativeHeight, height int) point {
	pixel := point{
		axisPixel(p.x, a.x, nativeWidth, nativeWidth),
		axisPixel(p.y, a.y, nativeHeight, nativeHeight),
	}
	return sceneViewport(nativeWidth, nativeHeight, height).scenePoint(pixel, height)
}

type action uint8

const (
	noAction action = iota
	flapAction
	exitAction
)

type bounds struct{ x, y, w, h int }

func (b bounds) contains(p point) bool {
	return p.x >= b.x && p.y >= b.y && p.x < b.x+b.w && p.y < b.y+b.h
}

func exitBounds() bounds { return bounds{sceneWidth - 56, 0, 56, 56} }

func (g *game) resultsTop() int {
	return (g.floor - scoreBoardHeight - 7 - playButtonHeight) / 2
}

func (g *game) startBounds() bounds {
	y := g.floor * 64 / 100
	if g.state == results {
		y = g.resultsTop() + scoreBoardHeight + 7
	}
	return bounds{(sceneWidth - playButtonWidth) / 2, y, playButtonWidth, playButtonHeight}
}

func (g *game) touchAction(p point, height int) action {
	if !(bounds{0, 0, sceneWidth, height}).contains(p) {
		return noAction
	}
	if exitBounds().contains(p) {
		return exitAction
	}
	if g.state != title && g.state != results {
		return flapAction
	}
	if g.canStart() && g.startBounds().contains(p) {
		return flapAction
	}
	return noAction
}
