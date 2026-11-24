package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// These units and transitions mirror flappybird.c. One simulation unit
// corresponds to two screen pixels on the original 480-pixel-wide panel.
const (
	sceneWidth            = 240
	floorHeight           = 112
	readyOffset           = (49*4 + 16 + 34) / 2
	gateWidth             = 52
	gateHeight            = 320
	gateGap               = 96
	gateSpacing           = 157
	gateMargin            = 32
	gapChange             = 24
	worldSpeed            = 2
	birdLeft              = 62
	birdBox               = 20
	lostAbove             = -22
	flapSpeed     float32 = -5.0
	gravity       float32 = 0.3
	maxFall       float32 = 8.0
	rotationFlap  float32 = -10.0
	rotationAccel float32 = 0.4
	rotationMin   float32 = -20.0
	rotationMax   float32 = 90.0
	overBoardAt           = 54
	overCountAt           = 78
	overDoneAt            = 123
	fadeSteps             = 10
	tickPeriod            = 16 * time.Millisecond
)

type phase uint8

const (
	title phase = iota
	ready
	flying
	paused
	falling
	results
)

type wingState uint8

const (
	wingsIdle wingState = iota
	wingsFlap
	wingsRest
)

type gate struct {
	x, gapTop int
	passed    bool
}

type game struct {
	state                                phase
	floor                                int
	birdY                                int
	speed, acceleration, angle, turn     float32
	gates                                [3]gate
	score, best, bestBefore              int
	scroll, background, bobDegrees       int
	wings                                wingState
	wingMS, wingIndex                    int
	overTick, fadeTick                   int
	changeToReady, keepLook, bestPending bool
	birdLook, backgroundLook             int
	rng                                  uint64
}

func newGame(height, best int, seed int64) *game {
	g := &game{
		floor: (height/2)*2 - floorHeight, best: best,
		rng: uint64(uint32(seed) - 1), keepLook: true, fadeTick: fadeSteps,
	}
	g.advanceLook()
	return g
}

// Compás uses musl rand(). The look and the gates share its sequence.
func (g *game) rand() int {
	g.rng = g.rng*6364136223846793005 + 1
	return int(g.rng >> 33)
}

func (g *game) advanceLook() {
	weights := [...]int{10, 10, 10, 10, 10, 10, 3, 1}
	pick := g.rand() % 64
	for color, weight := range weights {
		if pick < weight {
			g.birdLook = color
			break
		}
		pick -= weight
	}
	g.backgroundLook = g.rand() % 2
}

func (g *game) nextGap(previous int) int {
	high := g.floor - gateMargin - gateGap
	if high <= gateMargin {
		return gateMargin
	}
	var top int
	for i := 0; i < 8; i++ {
		top = gateMargin + g.rand()%(high-gateMargin+1)
		if previous < 0 || top <= previous-gapChange || top >= previous+gapChange {
			break
		}
	}
	return top
}

func (g *game) setWings(w wingState) {
	g.wings, g.wingMS, g.wingIndex = w, 0, 0
}

func (g *game) tickWings() {
	if g.wings == wingsRest {
		return
	}
	g.wingMS += 15
	threshold := 100
	if g.wings == wingsFlap {
		// The original threshold is 1000/30 ms. The timer advances by 15 ms.
		threshold = 34
	}
	if g.wingMS >= threshold {
		g.wingMS = 0
		g.wingIndex++
		if g.wings == wingsFlap && g.wingIndex >= 12 {
			g.wings = wingsRest
		}
	}
}

func (g *game) wingFrame() int {
	if g.wings == wingsRest {
		return 1
	}
	return [...]int{0, 1, 2, 1}[g.wingIndex%4]
}

func (g *game) readyY() int { return g.floor - readyOffset - birdBox/2 }

func (g *game) prepare() {
	if !g.keepLook {
		g.advanceLook()
	}
	g.keepLook = false
	g.state = ready
	g.birdY = g.readyY()
	g.score, g.overTick, g.bobDegrees = 0, 0, 0
	g.speed, g.acceleration, g.angle, g.turn = 0, 0, 0, 0
	g.setWings(wingsIdle)
	previous := -1
	for i := range g.gates {
		top := g.nextGap(previous)
		g.gates[i] = gate{x: sceneWidth + 120 + i*gateSpacing, gapTop: top}
		previous = top
	}
}

func (g *game) bob() float32 {
	radians := float32(g.bobDegrees) * float32(math.Pi) / 180
	return float32(math.Sin(float64(radians))) * 4
}

func (g *game) flap() {
	if g.birdY >= 0 {
		g.speed, g.acceleration, g.turn = flapSpeed, gravity, rotationFlap
		g.setWings(wingsFlap)
	}
}

func (g *game) canStart() bool {
	return g.fadeTick < 0 && (g.state == title || (g.state == results && g.overTick >= overCountAt))
}

func (g *game) action() {
	if g.fadeTick >= 0 {
		return
	}
	if g.canStart() {
		g.fadeTick, g.changeToReady = 0, true
		return
	}
	switch g.state {
	case ready:
		g.birdY = g.readyY() + int(math.Round(float64(g.bob())))
		g.state = flying
		g.flap()
	case paused:
		g.state = flying
		g.flap()
	case flying:
		g.flap()
	}
}

func (g *game) moveGates() {
	for i := range g.gates {
		g.gates[i].x -= worldSpeed
	}
	for i := range g.gates {
		p := &g.gates[i]
		if p.x+gateWidth < 0 {
			last := g.gates[0]
			for _, candidate := range g.gates[1:] {
				if candidate.x > last.x {
					last = candidate
				}
			}
			*p = gate{x: last.x + gateSpacing, gapTop: g.nextGap(last.gapTop)}
		}
		if !p.passed && p.x <= birdLeft {
			p.passed = true
			g.score++
		}
	}
}

func (g *game) hitGate() bool {
	for _, p := range g.gates {
		horizontal := birdLeft+birdBox >= p.x && birdLeft <= p.x+gateWidth
		upper := g.birdY+birdBox >= p.gapTop-gateHeight && g.birdY <= p.gapTop
		lower := g.birdY+birdBox >= p.gapTop+gateGap && g.birdY <= p.gapTop+gateGap+gateHeight
		if horizontal && (upper || lower) {
			return true
		}
	}
	return false
}

func (g *game) onGround() bool { return g.birdY >= g.floor-birdBox }

func (g *game) fall() {
	g.speed = min(g.speed+g.acceleration, maxFall)
	g.birdY = int(float32(g.birdY) + g.speed)
	if g.birdY > g.floor-birdBox {
		g.birdY = g.floor - birdBox
		g.speed, g.acceleration = 0, 0
	}
	g.angle = min(max(g.angle+g.turn, rotationMin), rotationMax)
	g.turn += rotationAccel
}

func (g *game) hit() {
	g.overTick = 0
	g.bestBefore = g.best
	if g.score > g.best {
		g.best, g.bestPending = g.score, true
	}
	g.state = falling
	if g.onGround() {
		g.state = results
	}
}

// step returns true when the results board has finished counting a new best.
func (g *game) step() bool {
	if g.state == title || g.state == ready || g.state == flying {
		g.scroll = (g.scroll + worldSpeed) % 336
		g.background = (g.background + 1) % 576
	}
	if g.state == title || g.state == ready {
		g.bobDegrees = (g.bobDegrees + 8) % 360
	}
	if g.state != paused && g.state != results {
		g.tickWings()
	}
	switch g.state {
	case flying:
		g.moveGates()
		g.fall()
		if g.onGround() || g.hitGate() || g.birdY < lostAbove {
			g.hit()
		}
	case falling:
		g.fall()
		if g.onGround() {
			g.state = results
		}
	}
	if g.state == results {
		g.overTick++
	}
	if g.fadeTick >= 0 {
		g.fadeTick++
		if g.fadeTick == fadeSteps && g.changeToReady {
			g.prepare()
			g.changeToReady = false
		}
		if g.fadeTick >= 2*fadeSteps {
			g.fadeTick = -1
		}
	}
	return g.state == results && g.overTick == overDoneAt && g.bestPending
}

// The original steps once per timer callback for regular and mildly late
// ticks. A longer gap catches up by up to four steps or pauses after a stall.
func stepsFor(elapsed time.Duration) int {
	us := elapsed.Microseconds()
	switch {
	case us > 250000:
		return 0
	case us >= 50000:
		return min(int((us+16667/2)/16667), 4)
	default:
		return 1
	}
}

// Match the panel's aspect ratio so a scene unit has the same size on both
// axes. Keep enough height for the ready bird on unusually short panels.
func playfieldHeight(width, height int) int {
	return max(360, height*sceneWidth/width)
}

const scoreName = "best-score.txt"

func loadBest(dir string) int {
	if dir == "" {
		return 0
	}
	b, err := os.ReadFile(filepath.Join(dir, scoreName))
	if err != nil {
		return 0
	}
	best, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || best < 0 {
		return 0
	}
	return best
}

func saveBest(dir string, best int) error {
	if dir == "" {
		return nil
	}
	f, err := os.CreateTemp(dir, ".best-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = fmt.Fprintln(f, best)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), filepath.Join(dir, scoreName))
}
