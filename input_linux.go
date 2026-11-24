package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type linuxInputEvent struct {
	Time       syscall.Timeval
	Type, Code uint16
	Value      int32
}

type input struct {
	fd    int
	touch touchState
}

func inputReadRequest(command int, size uintptr) uintptr {
	// MIPS has a three-bit ioctl direction field.
	direction := uintptr(0x80000000)
	switch runtime.GOARCH {
	case "mips", "mipsle", "mips64", "mips64le", "ppc64", "ppc64le":
		direction = 0x40000000
	}
	return direction | size<<16 | uintptr('E')<<8 | uintptr(command)
}

func readAbs(fd, code int) (a absInfo) {
	ioctl(fd, inputReadRequest(0x40+code, unsafe.Sizeof(a)), unsafe.Pointer(&a))
	return a
}

func readInputName(fd int) string {
	var name [256]byte
	if err := ioctl(fd, inputReadRequest(0x06, uintptr(len(name))), unsafe.Pointer(&name[0])); err != nil {
		return ""
	}
	return strings.TrimRight(string(name[:]), "\x00")
}

func openInputs(board string, width, height int) ([]input, error) {
	paths, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, err
	}
	var inputs []input
	for _, path := range paths {
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
		if err != nil {
			continue
		}
		slots := readAbs(fd, absSlot)
		name := readInputName(fd)
		in := input{fd: fd, touch: touchState{
			single:   (touchAxes{readAbs(fd, absX), readAbs(fd, absY)}).calibrated(board, name, width, height),
			multi:    (touchAxes{readAbs(fd, absMTX), readAbs(fd, absMTY)}).calibrated(board, name, width, height),
			hasSlots: slots.valid(), slotMin: slots.Min, slot: int(slots.Value - slots.Min),
		}}
		inputs = append(inputs, in)
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no readable input devices")
	}
	return inputs, nil
}

func keyAction(kind, code uint16, value int32) action {
	if kind != evKey || value != 1 {
		return noAction
	}
	switch code {
	case 1, 116, 158, 174: // Esc, Power, Back, Exit
		return exitAction
	case 28, 57, 115, 119, 163, 164, 207, 352: // Enter, Space, Volume Up, media keys, OK
		return flapAction
	}
	return noAction
}

func (in *input) drain(onKey func(action), onTouch func(contactPoint, touchAxes)) error {
	const size = int(unsafe.Sizeof(linuxInputEvent{}))
	var buf [size * 32]byte
	for batch := 0; batch < 4; batch++ {
		n, err := syscall.Read(in.fd, buf[:])
		if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EINTR) {
			return nil
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("input device disconnected")
		}
		for offset := 0; offset+size <= n; offset += size {
			data := buf[offset+size-8 : offset+size]
			kind, code := binary.LittleEndian.Uint16(data), binary.LittleEndian.Uint16(data[2:])
			value := int32(binary.LittleEndian.Uint32(data[4:]))
			if a := keyAction(kind, code, value); a != noAction {
				onKey(a)
			}
			if p, axes, pressed := in.touch.event(kind, code, value); pressed {
				onTouch(p, axes)
			}
		}
	}
	return nil
}
