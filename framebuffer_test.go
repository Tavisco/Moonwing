package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestFramebufferScaleAndOffsets(t *testing.T) {
	for _, panel := range []struct{ width, height, sceneHeight int }{
		{480, 800, 400}, {480, 720, 360}, {320, 480, 360}, {240, 320, 360}, {480, 320, 360},
	} {
		for _, depth := range []int{16, 24, 32} {
			for _, light := range []int{fadeSteps, fadeSteps / 2} {
				t.Run(fmt.Sprintf("%dx%d/%dbpp/light%d", panel.width, panel.height, depth, light), func(t *testing.T) {
					mode := fbVariable{Width: uint32(panel.width), Height: uint32(panel.height), X: 4, Bits: uint32(depth)}
					if depth == 16 {
						mode.Red, mode.Green, mode.Blue = bitfield{Offset: 11, Length: 5}, bitfield{Offset: 5, Length: 6}, bitfield{Length: 5}
					} else {
						mode.Red, mode.Green, mode.Blue = bitfield{Offset: 16, Length: 8}, bitfield{Offset: 8, Length: 8}, bitfield{Length: 8}
					}
					d := display{mode: mode, bytes: depth / 8, nativeRGB565: depth == 16}
					d.stride = (panel.width + 8) * d.bytes
					const pageOffset = 3
					d.memory = bytes.Repeat([]byte{0xa5}, (pageOffset+panel.height+1)*d.stride)
					d.palette[light] = newPixelPalette(mode, light)
					d.configure(panel.sceneHeight)
					c := newCanvas(panel.sceneHeight, nil)
					c.light = light
					for i := range c.pixels {
						c.pixels[i] = uint16(i%65535 + 1)
					}
					d.drawPage(c, pageOffset)
					for row := 0; row < len(d.memory)/d.stride; row++ {
						for column := 0; column < d.stride/d.bytes; column++ {
							y, x := row-pageOffset, column-int(mode.X)
							at := row*d.stride + column*d.bytes
							if y < 0 || y >= panel.height || x < 0 || x >= panel.width {
								for _, b := range d.memory[at : at+d.bytes] {
									if b != 0xa5 {
										t.Fatalf("wrote outside visible page at %d,%d", column, row)
									}
								}
								continue
							}
							var want uint32
							if d.view.contains(point{x, y}) {
								sx := (x - d.view.x) * sceneWidth / d.view.w
								sy := (y - d.view.y) * c.height / d.view.h
								want = d.palette[light].word(c.pixels[sy*sceneWidth+sx])
							}
							for b := 0; b < d.bytes; b++ {
								if got := d.memory[at+b]; got != byte(want>>uint(8*b)) {
									t.Fatalf("pixel %d,%d byte %d = %02x, want %02x", x, y, b, got, byte(want>>uint(8*b)))
								}
							}
						}
					}
				})
			}
		}
	}
}
