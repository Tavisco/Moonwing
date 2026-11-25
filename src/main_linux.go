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
	// Present before input discovery, score I/O, or PNG decoding. Reuse this
	// canvas for the menu so loading needs no assets or extra frame buffer.
	c := newCanvas(worldHeight, nil)
	c.drawLoading()
	d.present(c)

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
	c.art = art
	// Loading has already covered startup; show the ready menu immediately.
	g.fadeTick = -1
	c.draw(g)
	d.present(c)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(stop)
	last := time.Now()
	pacer := framePacer{period: d.mode.refreshPeriod(), next: last}
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-stop:
			return nil
		case <-timer.C:
		}
		now := time.Now()
		advance := shouldAdvanceFrame(now.Sub(last))
		last = now
		oldState, oldFade := g.state, g.fadeTick
		wasAnimated := g.animated()
		if !advance && g.state == flying {
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
		if advance {
			if g.step() {
				save()
			}
		}
		animated := g.animated()
		// Draw the completed results once before presentation goes idle.
		changed := oldState != g.state || oldFade != g.fadeTick || (wasAnimated && advance)
		if changed {
			c.draw(g)
		}
		var flipWait time.Duration
		if changed || animated {
			flipWait = d.present(c)
		}
		finished := time.Now()
		pacer.finish(finished, flipWait)
		timer.Reset(max(0, time.Until(pacer.next)))
	}
}
