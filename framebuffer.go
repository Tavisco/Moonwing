package main

import "unsafe"

type bitfield struct {
	Offset, Length, Reverse uint32
}

// Linux fbdev UAPI. The unsigned longs in the fixed structure follow the
// userspace ABI. All members of the variable structure are 32 bits.
type fbVariable struct {
	Width, Height, VirtualWidth, VirtualHeight               uint32
	X, Y, Bits, Gray                                         uint32
	Red, Green, Blue, Alpha                                  bitfield
	Nonstandard, Activate, MillimetersH, MillimetersW, Accel uint32
	PixelClock, Left, Right, Upper, Lower, HSync, VSync      uint32
	Sync, VMode, Rotate, ColorSpace                          uint32
	Reserved                                                 [4]uint32
}

type fbFixed struct {
	ID                               [16]byte
	MemoryStart                      uintptr
	MemoryLen, Type, TypeAux, Visual uint32
	XPan, YPan, YWrap                uint16
	Stride                           uint32
	MMIOStart                        uintptr
	MMIOLen, Accel                   uint32
	Capabilities                     uint16
	Reserved                         [2]uint16
}

type display struct {
	fd                  int
	mode                fbVariable
	stride, bytes, back int
	memory              []byte
	palette             [fadeSteps + 1]pixelPalette
	panActivate         uint32
	nativeRGB565        bool
	view                bounds
	sourceX             []int
	rowWords            []uint32 // aligned, reusable RAM staging for a full output row
}

func (d *display) configure(height int) {
	d.view = sceneViewport(int(d.mode.Width), int(d.mode.Height), height)
	d.rowWords = make([]uint32, (int(d.mode.Width)*d.bytes+3)/4)
	d.sourceX = make([]int, d.view.w)
	for x := range d.sourceX {
		d.sourceX[x] = x * sceneWidth / d.view.w
	}
}

// Convert the RGB565 canvas to the framebuffer's channel layout. Each fade
// level needs only 128 channel entries, so conversion also stays small on MIPS.
type pixelPalette struct {
	red, blue [32]uint32
	green     [64]uint32
	alpha     uint32
}

func newPixelPalette(v fbVariable, light int) pixelPalette {
	p := pixelPalette{alpha: channel(255, v.Alpha)}
	for i := range p.red {
		value := uint8((i*255*light + 31*fadeSteps/2) / (31 * fadeSteps))
		p.red[i] = channel(value, v.Red)
		p.blue[i] = channel(value, v.Blue)
	}
	for i := range p.green {
		value := uint8((i*255*light + 63*fadeSteps/2) / (63 * fadeSteps))
		p.green[i] = channel(value, v.Green)
	}
	return p
}

func (p *pixelPalette) word(rgb uint16) uint32 {
	return p.red[rgb>>11] | p.green[(rgb>>5)&63] | p.blue[rgb&31] | p.alpha
}

func channel(value uint8, field bitfield) uint32 {
	val := (uint32(value)*((1<<field.Length)-1) + 127) / 255
	if field.Reverse != 0 {
		var reversed uint32
		for i := uint32(0); i < field.Length; i++ {
			reversed = reversed<<1 | (val & 1)
			val >>= 1
		}
		val = reversed
	}
	return val << field.Offset
}

func (d *display) drawPage(c *canvas, offset int) {
	width, height := int(d.mode.Width), int(d.mode.Height)
	rowBytes := width * d.bytes
	start := offset*d.stride + int(d.mode.X)*d.bytes
	palette := &d.palette[c.light]
	direct := c.light == fadeSteps && d.nativeRGB565
	left, right := d.view.x*d.bytes, (d.view.x+d.view.w)*d.bytes
	paired := d.bytes == 2 && d.view.w == 2*sceneWidth && left%4 == 0
	staged := unsafe.Slice((*byte)(unsafe.Pointer(&d.rowWords[0])), rowBytes)
	clear(staged)
	row := staged[left:right]
	previousY := -1
	for y := 0; y < height; y++ {
		at := start + y*d.stride
		destination := d.memory[at : at+rowBytes]
		if y < d.view.y || y >= d.view.y+d.view.h {
			clear(destination)
			continue
		}
		sourceY := (y - d.view.y) * c.height / d.view.h
		if sourceY == previousY {
			copy(destination, staged)
			continue
		}
		source := c.pixels[sourceY*sceneWidth : (sourceY+1)*sceneWidth]
		switch {
		case paired && direct:
			pairs := unsafe.Slice((*uint32)(unsafe.Pointer(&row[0])), sceneWidth)
			for x, color := range source {
				pixel := uint32(color)
				pairs[x] = pixel | pixel<<16
			}
		case paired:
			pairs := unsafe.Slice((*uint32)(unsafe.Pointer(&row[0])), sceneWidth)
			for x, color := range source {
				pixel := palette.word(color)
				pairs[x] = pixel | pixel<<16
			}
		case direct:
			words := unsafe.Slice((*uint16)(unsafe.Pointer(&row[0])), d.view.w)
			if d.view.w == sceneWidth {
				copy(words, source)
			} else {
				for x, sx := range d.sourceX {
					words[x] = source[sx]
				}
			}
		case d.bytes == 2:
			words := unsafe.Slice((*uint16)(unsafe.Pointer(&row[0])), d.view.w)
			for x, sx := range d.sourceX {
				words[x] = uint16(palette.word(source[sx]))
			}
		case d.bytes == 3:
			for x, sx := range d.sourceX {
				pixel := palette.word(source[sx])
				i := x * 3
				row[i], row[i+1], row[i+2] = byte(pixel), byte(pixel>>8), byte(pixel>>16)
			}
		case d.bytes == 4:
			words := unsafe.Slice((*uint32)(unsafe.Pointer(&row[0])), d.view.w)
			for x, sx := range d.sourceX {
				words[x] = palette.word(source[sx])
			}
		}
		// Both the first row and its vertical duplicates are copied from
		// cached RAM. Never read previously written framebuffer pixels.
		copy(destination, staged)
		previousY = sourceY
	}
}
