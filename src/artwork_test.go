package main

import (
	"bytes"
	"image/png"
	"testing"
)

func TestBitmapTextUsesWholeArtworkPixels(t *testing.T) {
	art, err := loadArtwork()
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []struct {
		name string
		img  sprite
	}{
		{"logo.png", art.logo}, {"get-ready.png", art.ready}, {"how-to.png", art.howTo},
		{"game-over.png", art.over}, {"play-button.png", art.play}, {"score-board.png", art.board},
		{"exit-button.png", art.exit}, {"new-record-label.png", art.record},
	} {
		data, err := builtInArtwork.ReadFile("artwork/" + label.name)
		if err != nil {
			t.Fatal(err)
		}
		size, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if label.img.width%size.Width != 0 || label.img.height%size.Height != 0 ||
			label.img.width/size.Width != label.img.height/size.Height {
			t.Errorf("%s resized from %dx%d to %dx%d: lettering needs the same whole-pixel scale on both axes",
				label.name, size.Width, size.Height, label.img.width, label.img.height)
		}
	}
}

func TestEmbeddedPlayButtonRenders(t *testing.T) {
	art, err := loadArtwork()
	if err != nil {
		t.Fatal(err)
	}
	g := newGame(360, 0, 1)
	c := newCanvas(360, art)
	c.draw(g)

	button := &art.play
	area := g.startBounds()
	if button.width != area.w || button.height != area.h {
		t.Fatal("the displayed button and its touch target must have the same size")
	}
	background := &art.backgrounds[g.backgroundLook]
	for row := 0; row < button.height; row++ {
		y := area.y + row
		srcY := background.height - floorHeight - g.floor + y
		for col := 0; col < button.width; col++ {
			x := area.x + col
			bg := background.pixels[srcY*background.width+x]
			if background.alpha != nil {
				bg = blend565(0, bg, background.alpha[srcY*background.width+x])
			}
			i := row*button.width + col
			want := button.pixels[i]
			if button.alpha != nil {
				want = blend565(bg, want, button.alpha[i])
			}
			for sub := 0; sub < 2; sub++ {
				if got := c.pixels[y*rasterWidth+x*2+sub]; got != want {
					t.Fatalf("button pixel %d,%d = %04x, want %04x", col, row, got, want)
				}
			}
		}
	}
}
