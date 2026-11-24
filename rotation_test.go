package main

import (
	"math"
	"testing"
)

func TestRotationLookupAccuracy(t *testing.T) {
	for hundredths := -2000; hundredths <= 9000; hundredths++ {
		angle := float32(hundredths) / 100
		sin, cos := rotationCoefficients(angle)
		s, c := math.Sincos(float64(angle) * math.Pi / 180)
		wantSin, wantCos := int(math.Round(s*1024)), int(math.Round(c*1024))
		if delta := sin - wantSin; delta < -1 || delta > 1 {
			t.Fatalf("angle %v: sine %d, want %d", angle, sin, wantSin)
		}
		if delta := cos - wantCos; delta < -1 || delta > 1 {
			t.Fatalf("angle %v: cosine %d, want %d", angle, cos, wantCos)
		}
	}
}

// The previous inverse-rotation formula, kept as an independent pixel-level
// reference for clipping, transparency, and the incremental-coordinate loop.
func referenceBird(c *canvas, s *sprite, x, y int, angle float32) {
	sine, cosine := math.Sincos(float64(angle) * math.Pi / 180)
	sin, cos := int(math.Round(sine*1024)), int(math.Round(cosine*1024))
	raster := s.rasterSprite()
	const radius = 21
	for dy := max(-radius, -y); dy <= radius && y+dy < c.height; dy++ {
		for dx := max(-radius, -x); dx <= radius && x+dx < sceneWidth; dx++ {
			sx := ((dx*cos + dy*sin + 512) >> 10) + s.width/2
			sy := ((dy*cos - dx*sin + 512) >> 10) + s.height/2
			if sx < 0 || sx >= s.width || sy < 0 || sy >= s.height {
				continue
			}
			pixel := (y+dy)*rasterWidth + (x+dx)*2
			raster.blitRow(c.pixels[pixel:pixel+2], (sy*s.width+sx)*2)
		}
	}
}

func TestCachedBirdRotationMatchesReference(t *testing.T) {
	art, err := loadArtwork()
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range art.birds[0] {
		s := sprite{width: original.width, height: original.height,
			pixels: original.pixels, alpha: append([]uint8(nil), original.alpha...)}
		// Include translucent edges over different left/right background
		// samples, as well as the artwork's fully opaque/transparent pixels.
		for i := range s.alpha {
			if s.alpha[i] == 255 && i%3 == 0 {
				s.alpha[i] = 128
			}
		}
		got, want := newCanvas(360, nil), newCanvas(360, nil)
		for _, angle := range []float32{-20, -10, 0, 15, 45, 90} {
			for _, position := range []point{{120, 140}, {0, 0}, {239, 359}, {119, -10}} {
				for i := range got.pixels {
					got.pixels[i], want.pixels[i] = uint16(i), uint16(i)
				}
				got.bird(&s, position.x, position.y, angle)
				wantAngle := angle
				if !birdRotationEnabled {
					wantAngle = 0
				}
				referenceBird(want, &s, position.x, position.y, wantAngle)
				for i := range got.pixels {
					if got.pixels[i] != want.pixels[i] {
						t.Fatalf("angle %v, position %v, raster pixel %d: %04x, want %04x", angle, position, i, got.pixels[i], want.pixels[i])
					}
				}
			}
		}
		if allocations := testing.AllocsPerRun(10, func() { got.bird(&s, 120, 140, 12.3) }); allocations != 0 {
			t.Fatalf("rotation allocated %v times per frame", allocations)
		}
	}
}

func BenchmarkBirdRotation(b *testing.B) {
	art, err := loadArtwork()
	if err != nil {
		b.Fatal(err)
	}
	c := newCanvas(360, art)
	s := &art.birds[0][0]
	b.Run("cached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.bird(s, 120, 140, -10)
		}
	})
	b.Run("reference", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			referenceBird(c, s, 120, 140, -10)
		}
	})
}
