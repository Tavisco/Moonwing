package main

import "testing"

func TestPanelScaleAndTouchAgree(t *testing.T) {
	for _, panel := range []struct{ width, height, worldHeight int }{
		{480, 800, 400}, {480, 720, 360}, {320, 480, 360}, {240, 320, 360},
		{480, 320, 360}, // a short panel gets margins
	} {
		h := playfieldHeight(panel.width, panel.height)
		if h != panel.worldHeight {
			t.Errorf("%dx%d: height %d, want %d", panel.width, panel.height, h, panel.worldHeight)
		}
		v := sceneViewport(panel.width, panel.height, h)
		difference := v.w*h - v.h*sceneWidth
		if difference <= -h || difference >= h {
			t.Errorf("%dx%d distorts scene: viewport %v", panel.width, panel.height, v)
		}
		g := newGame(h, 0, 1)
		g.fadeTick = -1
		axes := touchAxes{absInfo{Max: int32(panel.width - 1)}, absInfo{Max: int32(panel.height - 1)}}
		for _, button := range []struct {
			area bounds
			want action
		}{{exitBounds(), exitAction}, {g.startBounds(), flapAction}} {
			p := point{button.area.x + button.area.w/2, button.area.y + button.area.h/2}
			raw := contactPoint{x: int32(v.x + p.x*v.w/sceneWidth), y: int32(v.y + p.y*v.h/h)}
			if got := g.touchAction(axes.screenPoint(raw, panel.width, panel.height, h), h); got != button.want {
				t.Errorf("%dx%d: visible button %v produced %v", panel.width, panel.height, button.area, got)
			}
		}
		if v.x > 0 {
			g.state = flying
			p := axes.screenPoint(contactPoint{x: 0, y: int32(panel.height / 2)}, panel.width, panel.height, h)
			if got := g.touchAction(p, h); got != noAction {
				t.Errorf("tap in margin produced %v", got)
			}
		}
	}
}

func TestBackgroundCropsInsteadOfStretching(t *testing.T) {
	bg := sprite{width: 288, height: 512, pixels: make([]uint16, 288*512)}
	for y := 0; y < bg.height; y++ {
		for x := 0; x < bg.width; x++ {
			bg.pixels[y*bg.width+x] = uint16(y*100 + x%100)
		}
	}
	for _, height := range []int{320, 360, 400, 600} {
		c := newCanvas(height, nil)
		floor := newGame(height, 0, 1).floor
		top := bg.height - floorHeight - floor
		c.tile(&bg, 73, top, 0, floor)
		for y := 0; y < floor; y++ {
			for x := 0; x < sceneWidth; x++ {
				want := bg.pixels[max(0, top+y)*bg.width+(x+73)%bg.width]
				if got := c.pixels[y*sceneWidth+x]; got != want {
					t.Fatalf("height %d, background %d,%d: %04x, want %04x", height, x, y, got, want)
				}
			}
		}
	}
}
