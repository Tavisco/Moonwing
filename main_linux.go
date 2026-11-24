package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Moonwing:", err)
		os.Exit(1)
	}
}

func run() error {
	d, err := openDisplay()
	if err != nil {
		return err
	}
	defer d.close()
	width, height := int(d.mode.Width), int(d.mode.Height)
	panelWidth, xErr := strconv.Atoi(os.Getenv("COMPAS_SCREEN_WIDTH"))
	panelHeight, yErr := strconv.Atoi(os.Getenv("COMPAS_SCREEN_HEIGHT"))
	if xErr != nil || yErr != nil || panelWidth < 96 || panelHeight < 96 ||
		panelWidth > 4096 || panelHeight > 4096 {
		panelWidth, panelHeight = width, height
	}
	worldHeight := playfieldHeight(panelWidth, panelHeight)
	d.configure(worldHeight)
	inputs, err := openInputs(os.Getenv("COMPAS_BOARD"), panelWidth, panelHeight)
	if err != nil {
		return err
	}
	defer func() {
		for _, in := range inputs {
			syscall.Close(in.fd)
		}
	}()
	scoreDir := os.Getenv("COMPAS_APP_DATA")
	g := newGame(worldHeight, loadBest(scoreDir), time.Now().UnixMicro())
	save := func() {
		if g.bestPending {
			if err := saveBest(scoreDir, g.best); err != nil {
				fmt.Fprintln(os.Stderr, "Moonwing score:", err)
				return
			}
			g.bestPending = false
		}
	}
	defer save()
	art, err := loadArtwork()
	if err != nil {
		return err
	}
	c := newCanvas(worldHeight, art)
	c.draw(g)
	d.present(c)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(stop)
	ticker := time.NewTicker(tickPeriod)
	defer ticker.Stop()
	last := time.Now()
	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
		}
		now := time.Now()
		steps := stepsFor(now.Sub(last))
		last = now
		oldState, oldFade := g.state, g.fadeTick
		if steps == 0 && g.state == flying {
			g.state = paused
		}

		quitting := false
		handleAction := func(a action) {
			if a == exitAction {
				quitting = true
			} else if a == flapAction && !quitting {
				g.action()
			}
		}
		handleTouch := func(p contactPoint, axes touchAxes) {
			handleAction(g.touchAction(axes.screenPoint(p, panelWidth, panelHeight, worldHeight), worldHeight))
		}
		for i := 0; i < len(inputs); {
			in := &inputs[i]
			if err := in.drain(handleAction, handleTouch); err != nil {
				fmt.Fprintln(os.Stderr, "Moonwing input:", err)
				syscall.Close(in.fd)
				inputs = append(inputs[:i], inputs[i+1:]...)
			} else {
				i++
			}
		}
		if quitting {
			return nil
		}
		if len(inputs) == 0 {
			return fmt.Errorf("all input devices disconnected")
		}
		for i := 0; i < steps; i++ {
			if g.step() {
				save()
			}
		}
		animated := g.state != paused && (g.state != results || g.overTick <= overDoneAt)
		if oldState != g.state || oldFade != g.fadeTick || (animated && steps > 0) {
			c.draw(g)
			d.present(c)
		}
	}
}
