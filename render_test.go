package main

import (
	"encoding/binary"
	"image"
	"testing"
)

func TestLoadingTitleMatchesVisibleMenuLettering(t *testing.T) {
	art, err := loadArtwork()
	if err != nil {
		t.Fatal(err)
	}
	for _, panel := range [][2]int{{480, 800}, {480, 720}, {320, 480}, {240, 320}} {
		width, height := panel[0], panel[1]
		worldHeight := playfieldHeight(width, height)
		loading, menu := newCanvas(worldHeight, nil), newCanvas(worldHeight, nil)
		loading.drawLoading() // Deliberately has no artwork available.
		menu.light = fadeSteps
		menu.centered(&art.logo, titleLogoBounds(worldHeight).y)
		d := display{mode: fbVariable{Width: uint32(width), Height: uint32(height)},
			bytes: 2, stride: width * 2, nativeRGB565: true, memory: make([]byte, width*height*2)}
		d.configure(worldHeight)
		letterBounds := func(c *canvas) image.Rectangle {
			d.drawPage(c, 0)
			var area image.Rectangle
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					if binary.LittleEndian.Uint16(d.memory[(y*width+x)*2:]) == 0xff9a {
						area = area.Union(image.Rect(x, y, x+1, y+1))
					}
				}
			}
			return area
		}
		got, want := letterBounds(loading), letterBounds(menu)
		if want.Empty() || got != want {
			t.Errorf("%dx%d: loading lettering %v, menu lettering %v", width, height, got, want)
		}
	}
}

func TestBackgroundScrollPreservesSingleDevicePixels(t *testing.T) {
	art, err := loadArtwork()
	if err != nil {
		t.Fatal(err)
	}
	bg := sprite{width: 288, height: 512, pixels: make([]uint16, 288*512)}
	for y := 0; y < bg.height; y++ {
		for x := 0; x < bg.width; x++ {
			bg.pixels[y*bg.width+x] = uint16(x + 1)
		}
	}
	art.backgrounds[0] = bg
	// A translucent foreground pixel must blend over each distinct half-unit
	// background sample; shifting the finished frame would move this label.
	art.logo = sprite{width: 1, height: 1, pixels: []uint16{0xf800}, alpha: []uint8{128}}
	g := newGame(400, 0, 1)
	g.fadeTick, g.backgroundLook = -1, 0
	c := newCanvas(400, art)
	d := display{mode: fbVariable{Width: 480, Height: 800}, bytes: 2,
		stride: 960, nativeRGB565: true, memory: make([]byte, 960*800)}
	d.configure(400)
	for _, offset := range []int{0, 1, 2, 3, 574, 575, 0} {
		g.background = offset
		c.draw(g)
		d.drawPage(c, 0)
		for x := 0; x < 480; x++ {
			want := uint16(((x+offset)/2)%bg.width + 1)
			if got := binary.LittleEndian.Uint16(d.memory[x*2:]); got != want {
				t.Fatalf("scroll %d, output x=%d: %04x, want %04x", offset, x, got, want)
			}
		}
		x, y := (sceneWidth-art.logo.width)/2*2, g.floor*21/100*2
		for sub := 0; sub < 2; sub++ {
			background := uint16(((x+sub+offset)/2)%bg.width + 1)
			want := blend565(background, 0xf800, 128)
			if got := binary.LittleEndian.Uint16(d.memory[(y*480+x+sub)*2:]); got != want {
				t.Fatalf("foreground shifted or blended against wrong sample at scroll %d: %04x, want %04x", offset, got, want)
			}
		}
		area := g.startBounds()
		for row := 0; row < art.play.height; row++ {
			for col := 0; col < art.play.width; col++ {
				i := row*art.play.width + col
				if art.play.alpha != nil && art.play.alpha[i] != 255 {
					continue
				}
				for sub := 0; sub < 2; sub++ {
					at := ((area.y+row)*2*480 + (area.x+col)*2 + sub) * 2
					if got := binary.LittleEndian.Uint16(d.memory[at:]); got != art.play.pixels[i] {
						t.Fatalf("scroll %d moved visible Fly button at %d,%d", offset, col, row)
					}
				}
			}
		}
	}
}
