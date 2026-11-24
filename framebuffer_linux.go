package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	fbGetVariable = 0x4600
	fbGetFixed    = 0x4602
	fbPan         = 0x4606
)

func ioctl(fd int, request uintptr, data unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request, uintptr(data))
	if errno != 0 {
		return errno
	}
	return nil
}

func openDisplay() (*display, error) {
	fd, err := syscall.Open("/dev/fb0", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open framebuffer: %w", err)
	}
	d, err := setupDisplay(fd)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}
	return d, nil
}

func setupDisplay(fd int) (*display, error) {
	d := &display{fd: fd, back: -1}
	var fixed fbFixed
	if err := ioctl(fd, fbGetVariable, unsafe.Pointer(&d.mode)); err != nil {
		return nil, fmt.Errorf("read framebuffer mode: %w", err)
	}
	if err := ioctl(fd, fbGetFixed, unsafe.Pointer(&fixed)); err != nil {
		return nil, fmt.Errorf("read framebuffer layout: %w", err)
	}
	v := d.mode
	d.bytes, d.stride = int(v.Bits/8), int(fixed.Stride)
	validDepth := v.Bits == 16 || v.Bits == 24 || v.Bits == 32
	if !validDepth || fixed.Type != 0 || (fixed.Visual != 2 && fixed.Visual != 4) ||
		v.Width < 96 || v.Height < 96 || v.Width > 4096 || v.Height > 4096 ||
		uint64(v.X)+uint64(v.Width) > uint64(fixed.Stride)/uint64(max(1, d.bytes)) ||
		d.stride%d.bytes != 0 && d.bytes != 3 ||
		uint64(fixed.Stride)*(uint64(v.Y)+uint64(v.Height)) > uint64(fixed.MemoryLen) ||
		fixed.MemoryLen > 128*1024*1024 {
		return nil, fmt.Errorf("unsupported framebuffer layout: %dx%d at %d bits", v.Width, v.Height, v.Bits)
	}
	fields := [...]bitfield{v.Red, v.Green, v.Blue, v.Alpha}
	for i, field := range fields {
		if field.Length > 16 || field.Offset > v.Bits || field.Length > v.Bits-field.Offset ||
			(i < 3 && field.Length == 0) {
			return nil, fmt.Errorf("unsupported framebuffer channel layout")
		}
	}
	d.nativeRGB565 = v.Bits == 16 && v.Red == (bitfield{Offset: 11, Length: 5}) &&
		v.Green == (bitfield{Offset: 5, Length: 6}) && v.Blue == (bitfield{Length: 5}) &&
		v.Alpha.Length == 0
	for light := range d.palette {
		d.palette[light] = newPixelPalette(v, light)
	}
	m, err := syscall.Mmap(fd, 0, int(fixed.MemoryLen), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("map framebuffer: %w", err)
	}
	d.memory = m
	height, current := int(v.Height), int(v.Y)
	if int(v.VirtualHeight) >= 2*height && len(m)/d.stride >= 2*height {
		switch {
		case current >= height:
			d.back = 0
		case current == 0 && (fixed.YPan == 0 || height%int(fixed.YPan) == 0):
			d.back = height
		}
	}
	return d, nil
}

func (d *display) close() {
	syscall.Munmap(d.memory)
	syscall.Close(d.fd)
}

func (d *display) present(c *canvas) {
	if d.back < 0 {
		d.drawPage(c, int(d.mode.Y))
		return
	}
	d.drawPage(c, d.back)
	next := d.mode
	next.Y = uint32(d.back)
	next.Activate = 0
	if err := ioctl(d.fd, fbPan, unsafe.Pointer(&next)); err != nil {
		d.back = -1
		d.drawPage(c, int(d.mode.Y))
		return
	}
	d.back = int(d.mode.Y)
	d.mode.Y = next.Y
}
