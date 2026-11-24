package main

import (
	"math"
	"strconv"
)

// The game remains 240 units wide. Two raster columns per unit retain the
// background's half-unit movement without changing sprite sizes or physics.
const rasterWidth = sceneWidth * 2

type canvas struct {
	height int
	pixels []uint16
	light  int
	art    *artwork
}

func newCanvas(height int, art *artwork) *canvas {
	return &canvas{height: height, pixels: make([]uint16, rasterWidth*height), art: art}
}

func (c *canvas) drawLoading() {
	const background uint16 = 0x10c4
	const foreground uint16 = 0xbd59
	c.light = fadeSteps
	for i := range c.pixels {
		c.pixels[i] = background
	}
	c.textInBounds("MOONWING", titleLetteringBounds(c.height), 0xff9a)
	c.center("LOADING...", sceneWidth/2, (c.height-14)/2, 2, foreground)
}

func (c *canvas) rect(x, y, w, h int, col uint16) {
	left, right := max(0, x)*2, min(sceneWidth, x+w)*2
	top, bottom := max(0, y), min(c.height, y+h)
	for row := top; row < bottom; row++ {
		for at := row*rasterWidth + left; at < row*rasterWidth+right; at++ {
			c.pixels[at] = col
		}
	}
}

func blend565(dst, src uint16, a uint8) uint16 {
	alpha, inverse := uint32(a), uint32(255-a)
	from, to := uint32(dst), uint32(src)
	r := ((from>>11)*inverse + (to>>11)*alpha + 127) / 255
	g := (((from>>5)&63)*inverse + ((to>>5)&63)*alpha + 127) / 255
	b := ((from&31)*inverse + (to&31)*alpha + 127) / 255
	return uint16(r<<11 | g<<5 | b)
}

func (s *sprite) blitRow(dst []uint16, from int) {
	pixels := s.pixels[from : from+len(dst)]
	if s.alpha == nil {
		copy(dst, pixels)
		return
	}
	for i, alpha := range s.alpha[from : from+len(dst)] {
		switch alpha {
		case 0:
		case 255:
			dst[i] = pixels[i]
		default:
			dst[i] = blend565(dst[i], pixels[i], alpha)
		}
	}
}

func (c *canvas) blit(s *sprite, x, y, bottom int) {
	x *= 2
	left, right := max(0, x), min(rasterWidth, x+s.width*2)
	if left >= right {
		return
	}
	s = s.rasterSprite()
	bottom = min(bottom, c.height, y+s.height)
	for destY := max(0, y); destY < bottom; destY++ {
		dst := c.pixels[destY*rasterWidth+left : destY*rasterWidth+right]
		s.blitRow(dst, (destY-y)*s.width+left-x)
	}
}

func (c *canvas) tile(s *sprite, startX, sourceTop, top, bottom int) {
	s = s.rasterSprite()
	startX %= s.width
	first := min(s.width-startX, rasterWidth)
	for y := top; y < bottom; y++ {
		// Crop the background at its native scale. Extend the sky's first
		// row on tall panels instead of stretching every row of the image.
		srcY := min(max(sourceTop+y-top, 0), s.height-1)
		dst := c.pixels[y*rasterWidth : (y+1)*rasterWidth]
		s.blitRow(dst[:first], srcY*s.width+startX)
		if first < rasterWidth {
			s.blitRow(dst[first:], srcY*s.width)
		}
	}
}

func (c *canvas) centered(s *sprite, y int) {
	c.blit(s, (sceneWidth-s.width)/2, y, c.height)
}

func (c *canvas) bird(s *sprite, x, y int, angle float32) {
	if !birdRotationEnabled || angle == 0 {
		c.blit(s, x-s.width/2, y-s.height/2, c.height)
		return
	}
	sin, cos := rotationCoefficients(angle)
	const radius = 21 // the half diagonal of a 34×24 sprite
	left, right := max(-radius, -x), min(radius, sceneWidth-1-x)
	for dy := max(-radius, -y); dy <= radius && y+dy < c.height; dy++ {
		sourceX, sourceY := left*cos+dy*sin+512, dy*cos-left*sin+512
		for dx := left; dx <= right; dx++ {
			sx := (sourceX >> 10) + s.width/2
			sy := (sourceY >> 10) + s.height/2
			if sx >= 0 && sx < s.width && sy >= 0 && sy < s.height {
				pixel := (y+dy)*rasterWidth + (x+dx)*2
				from := sy*s.width + sx
				color := s.pixels[from]
				if s.alpha == nil || s.alpha[from] == 255 {
					c.pixels[pixel], c.pixels[pixel+1] = color, color
				} else if alpha := s.alpha[from]; alpha != 0 {
					c.pixels[pixel] = blend565(c.pixels[pixel], color, alpha)
					c.pixels[pixel+1] = blend565(c.pixels[pixel+1], color, alpha)
				}
			}
			sourceX, sourceY = sourceX+cos, sourceY-sin
		}
	}
}

var glyphs = map[byte][7]uint8{
	'A': {14, 17, 17, 31, 17, 17, 17}, 'B': {30, 17, 17, 30, 17, 17, 30},
	'C': {14, 17, 16, 16, 16, 17, 14}, 'D': {30, 17, 17, 17, 17, 17, 30},
	'E': {31, 16, 16, 30, 16, 16, 31}, 'F': {31, 16, 16, 30, 16, 16, 16},
	'G': {14, 17, 16, 23, 17, 17, 15}, 'H': {17, 17, 17, 31, 17, 17, 17},
	'I': {31, 4, 4, 4, 4, 4, 31}, 'J': {7, 2, 2, 2, 18, 18, 12},
	'K': {17, 18, 20, 24, 20, 18, 17}, 'L': {16, 16, 16, 16, 16, 16, 31},
	'M': {17, 27, 21, 21, 17, 17, 17}, 'N': {17, 25, 21, 19, 17, 17, 17},
	'O': {14, 17, 17, 17, 17, 17, 14}, 'P': {30, 17, 17, 30, 16, 16, 16},
	'Q': {14, 17, 17, 17, 21, 18, 13}, 'R': {30, 17, 17, 30, 20, 18, 17},
	'S': {15, 16, 16, 14, 1, 1, 30}, 'T': {31, 4, 4, 4, 4, 4, 4},
	'U': {17, 17, 17, 17, 17, 17, 14}, 'V': {17, 17, 17, 17, 17, 10, 4},
	'W': {17, 17, 17, 21, 21, 21, 10}, 'X': {17, 17, 10, 4, 10, 17, 17},
	'Y': {17, 17, 10, 4, 4, 4, 4}, 'Z': {31, 1, 2, 4, 8, 16, 31},
	'0': {14, 17, 19, 21, 25, 17, 14}, '1': {4, 12, 4, 4, 4, 4, 14},
	'2': {14, 17, 1, 2, 4, 8, 31}, '3': {30, 1, 1, 14, 1, 1, 30},
	'4': {2, 6, 10, 18, 31, 2, 2}, '5': {31, 16, 16, 30, 1, 1, 30},
	'6': {14, 16, 16, 30, 17, 17, 14}, '7': {31, 1, 2, 4, 8, 8, 8},
	'8': {14, 17, 17, 14, 17, 17, 14}, '9': {14, 17, 17, 15, 1, 1, 14},
	'a': {0, 0, 14, 1, 15, 17, 15}, 'b': {16, 16, 30, 17, 17, 17, 30},
	'c': {0, 0, 14, 17, 16, 17, 14}, 'd': {1, 1, 15, 17, 17, 17, 15},
	'e': {0, 0, 14, 17, 31, 16, 14}, 'f': {6, 9, 8, 28, 8, 8, 8},
	'g': {0, 15, 17, 17, 15, 1, 14}, 'h': {16, 16, 30, 17, 17, 17, 17},
	'i': {4, 0, 12, 4, 4, 4, 14}, 'j': {2, 0, 6, 2, 2, 18, 12},
	'k': {16, 16, 18, 20, 24, 20, 18}, 'l': {12, 4, 4, 4, 4, 4, 14},
	'm': {0, 0, 26, 21, 21, 17, 17}, 'n': {0, 0, 30, 17, 17, 17, 17},
	'o': {0, 0, 14, 17, 17, 17, 14}, 'p': {0, 0, 30, 17, 30, 16, 16},
	'q': {0, 0, 15, 17, 15, 1, 1}, 'r': {0, 0, 22, 25, 16, 16, 16},
	's': {0, 0, 15, 16, 14, 1, 30}, 't': {8, 8, 30, 8, 8, 9, 6},
	'u': {0, 0, 17, 17, 17, 19, 13}, 'v': {0, 0, 17, 17, 17, 10, 4},
	'w': {0, 0, 17, 17, 21, 21, 10}, 'x': {0, 0, 17, 10, 4, 10, 17},
	'y': {0, 0, 17, 17, 15, 1, 14}, 'z': {0, 0, 31, 2, 4, 8, 31},
	'.': {0, 0, 0, 0, 0, 12, 12}, ':': {0, 12, 12, 0, 12, 12, 0},
	'/': {1, 2, 2, 4, 8, 8, 16}, '-': {0, 0, 0, 31, 0, 0, 0},
}

func (c *canvas) text(s string, x, y, scale int, col uint16) {
	for i := 0; i < len(s); i++ {
		for row, bits := range glyphs[s[i]] {
			for column := 0; column < 5; column++ {
				if bits&(1<<uint(4-column)) != 0 {
					c.rect(x+column*scale, y+row*scale, scale, scale, col)
				}
			}
		}
		x += 6 * scale
	}
}

// Fit built-in text to an existing layout box without decoding an image or
// allocating a temporary sprite. All glyphs are drawn directly as rectangles.
func (c *canvas) textInBounds(s string, area bounds, col uint16) {
	columns := 6*len(s) - 1
	for i := 0; i < len(s); i++ {
		for row, bits := range glyphs[s[i]] {
			for column := 0; column < 5; column++ {
				if bits&(1<<uint(4-column)) == 0 {
					continue
				}
				left := (6*i + column) * area.w / columns
				right := (6*i + column + 1) * area.w / columns
				top, bottom := row*area.h/7, (row+1)*area.h/7
				c.rect(area.x+left, area.y+top, right-left, bottom-top, col)
			}
		}
	}
}

func (c *canvas) center(s string, x, y, scale int, col uint16) {
	c.text(s, x-(6*len(s)-1)*scale/2, y, scale, col)
}

func (c *canvas) number(digits *[10]sprite, value, x, y int, centered bool) {
	text := strconv.Itoa(value)
	if centered {
		width := 0
		for i := range text {
			width += digits[text[i]-'0'].width
		}
		x -= width / 2
	}
	for i := range text {
		digit := &digits[text[i]-'0']
		c.blit(digit, x, y, c.height)
		x += digit.width
	}
}

func (c *canvas) draw(g *game) {
	const cream uint16 = 0xff9a
	const muted uint16 = 0xbd59
	const card uint16 = 0x18c6
	c.light = fadeSteps
	if g.fadeTick >= 0 {
		c.light = max(fadeSteps-g.fadeTick, g.fadeTick-fadeSteps)
	}

	a := c.art
	bg := &a.backgrounds[g.backgroundLook]
	if bg.alpha != nil || a.ground.alpha != nil {
		clear(c.pixels)
	}
	c.tile(bg, g.background, bg.height-floorHeight-g.floor, 0, g.floor)
	c.tile(&a.ground, g.scroll*2, 0, g.floor, c.height)
	if g.state >= flying {
		for _, p := range g.gates {
			c.blit(&a.upper, p.x, p.gapTop-gateHeight, g.floor)
			c.blit(&a.lower, p.x, p.gapTop+gateGap, g.floor)
		}
	}

	x, y := birdLeft+birdBox/2, g.birdY+birdBox/2
	switch g.state {
	case title:
		x, y = sceneWidth/2, g.floor*48/100+int(math.Round(float64(g.bob())))
	case ready:
		y += int(math.Round(float64(g.bob())))
	}
	c.bird(&a.birds[g.birdLook][g.wingFrame()], x, y, g.angle)

	switch g.state {
	case title:
		c.centered(&a.logo, titleLogoBounds(c.height).y)
		b := g.startBounds()
		c.blit(&a.play, b.x, b.y, c.height)
		c.center("BEST "+strconv.Itoa(g.best), sceneWidth/2, g.floor+24, 2, muted)
		const footerMargin = 8
		footerY := c.height - footerMargin - 7
		c.text("More at:", footerMargin, footerY-10, 1, muted)
		c.text("https://otavio.dev.br", footerMargin, footerY, 1, muted)
		c.text(appVersionLabel, sceneWidth-footerMargin-(6*len(appVersionLabel)-1), footerY, 1, muted)
	case ready:
		c.centered(&a.ready, g.floor*25/100)
		c.centered(&a.howTo, g.floor*68/100)
	case flying, falling:
		c.number(&a.bigScore, g.score, 14, 15, false)
	case paused:
		c.rect(25, g.floor*58/100, 190, 34, card)
		c.center("TAP TO RESUME", sceneWidth/2, g.floor*58/100+10, 2, cream)
	case results:
		c.drawResults(g)
	}

	b := exitBounds()
	c.blit(&a.exit, b.x+(b.w-a.exit.width)/2, b.y+(b.h-a.exit.height)/2, c.height)
}

func (c *canvas) drawResults(g *game) {
	a := c.art
	if g.overTick < overBoardAt {
		if g.overTick >= 24 {
			c.centered(&a.over, g.floor/2-45)
		}
		return
	}
	top := g.resultsTop()
	if g.overTick < overCountAt {
		top += (c.height - top) * (overCountAt - g.overTick) / (overCountAt - overBoardAt)
	}
	c.centered(&a.board, top)
	count := min(max(0, g.overTick-overCountAt), overDoneAt-overCountAt)
	shown := int(int64(g.score) * int64(count) / (overDoneAt - overCountAt))
	best := max(shown, g.bestBefore)
	c.number(&a.score, shown, sceneWidth/2-scoreBoardWidth/4, top+66, true)
	c.number(&a.score, best, sceneWidth/2+scoreBoardWidth/4, top+66, true)
	if g.overTick >= overDoneAt {
		if g.score >= 10 {
			medal := &a.medals[min(g.score/10-1, 2)]
			c.centered(medal, top+82)
		}
		if g.best > g.bestBefore {
			c.blit(&a.record, sceneWidth/2+scoreBoardWidth/4-a.record.width/2, top+94, c.height)
		}
	}
	if g.overTick >= overCountAt {
		b := g.startBounds()
		c.blit(&a.play, b.x, b.y, c.height)
	}
}
