package main

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"
	"image/png"
)

const (
	maxPNGDimension  = 1024
	playButtonWidth  = 104
	playButtonHeight = 58
	scoreBoardWidth  = 226
	scoreBoardHeight = 114
)

var birdNames = [...]string{"yellow", "red", "blue", "green", "peach", "purple", "cow", "rare"}

//go:embed artwork/*.png
var builtInArtwork embed.FS

type sprite struct {
	width, height int
	pixels        []uint16
	alpha         []uint8 // nil for opaque images
}

type artwork struct {
	backgrounds               [2]sprite
	birds                     [len(birdNames)][3]sprite
	ground, upper, lower      sprite
	logo, ready, howTo, over  sprite
	play, exit, board, record sprite
	medals                    [3]sprite
	score, bigScore           [10]sprite
}

func loadArtwork() (*artwork, error) {
	a := &artwork{}
	var firstError error
	load := func(name string, width, height int) sprite {
		if firstError != nil {
			return sprite{}
		}
		img, err := artworkSprite(name, width, height)
		if err != nil {
			firstError = err
		}
		return img
	}

	a.backgrounds[0] = load("BG-day.png", 288, 512)
	a.backgrounds[1] = load("BG-night.png", 288, 512)
	a.ground = load("ground.png", 336, floorHeight)
	a.upper = load("pipe-up.png", gateWidth, gateHeight)
	a.lower = load("pipe-down.png", gateWidth, gateHeight)

	for i, name := range birdNames {
		for frame := range a.birds[i] {
			a.birds[i][frame] = load(fmt.Sprintf("bird-%s-%02d.png", name, frame+1), 34, 24)
		}
	}
	for digit := range a.score {
		smallWidth, bigWidth := 7, 12
		if digit == 1 {
			smallWidth, bigWidth = 5, 8
		}
		a.score[digit] = load(fmt.Sprintf("score-%d.png", digit), smallWidth*2, 20)
		a.bigScore[digit] = load(fmt.Sprintf("score-game-%d.png", digit), bigWidth*2, 36)
	}
	for i, name := range [...]string{"bronze", "silver", "gold"} {
		a.medals[i] = load("medal-"+name+".png", 22, 22)
	}

	a.logo = load("logo.png", 178, 48)
	a.ready = load("get-ready.png", 184, 50)
	a.howTo = load("how-to.png", 57, 49)
	a.over = load("game-over.png", 192, 42)
	a.board = load("score-board.png", scoreBoardWidth, scoreBoardHeight)
	a.play = load("play-button.png", playButtonWidth, playButtonHeight)
	a.exit = load("exit-button.png", 28, 28)
	a.record = load("new-record-label.png", 32, 14)
	return a, firstError
}

func artworkSprite(name string, width, height int) (sprite, error) {
	data, err := builtInArtwork.ReadFile("artwork/" + name)
	if err == nil {
		img, decodeErr := decodeSprite(data, width, height)
		if decodeErr == nil {
			return img, nil
		}
		err = decodeErr
	}
	return sprite{}, fmt.Errorf("embedded artwork %s: %w", name, err)
}

func decodeSprite(b []byte, width, height int) (sprite, error) {
	reader := bytes.NewReader(b)
	dim, err := png.DecodeConfig(reader)
	if err != nil {
		return sprite{}, err
	}
	if dim.Width > maxPNGDimension || dim.Height > maxPNGDimension {
		return sprite{}, fmt.Errorf("PNG dimensions exceed %d pixels per side", maxPNGDimension)
	}
	reader.Reset(b)
	img, err := png.Decode(reader)
	if err != nil {
		return sprite{}, err
	}

	result := sprite{width: width, height: height, pixels: make([]uint16, width*height)}
	if opaque, ok := img.(interface{ Opaque() bool }); !ok || !opaque.Opaque() {
		result.alpha = make([]uint8, width*height)
	}
	for y := 0; y < height; y++ {
		srcY := y * dim.Height / height
		for x := 0; x < width; x++ {
			srcX := x * dim.Width / width
			px := color.NRGBAModel.Convert(img.At(srcX, srcY)).(color.NRGBA)
			i := y*width + x
			result.pixels[i] = rgb565(px.R, px.G, px.B)
			if result.alpha != nil {
				result.alpha[i] = px.A
			}
		}
	}
	return result, nil
}

func rgb565(r, g, b uint8) uint16 {
	return uint16(r>>3)<<11 | uint16(g>>2)<<5 | uint16(b>>3)
}
