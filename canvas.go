package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var SimulationCompleted = errors.New("Simulation completed")

type SimulatedAnnealingEngine interface {
	Iterate() error
	ToPixels() []byte
	GetSnapshot() image.Image
}

// Canvas handles the canvas visualization
type Canvas struct {

	// resolution of the canvas
	width  int
	height int

	gameRunning bool

	simulatedAnnealing SimulatedAnnealingEngine
	simulationDuration time.Duration
	simulationStart    time.Time
	lastSnapshot       time.Time
	snapshotsInterval  time.Duration
}

// NewCanvas creates a canvas with the simulated annealing ready to start
func NewCanvas(
	width int,
	height int,
	simulatedAnnealing SimulatedAnnealingEngine,
	simulationDuration time.Duration,
	snapshotsInterval time.Duration,
) (*Canvas, error) {

	g := &Canvas{
		width:              width,
		height:             height,
		gameRunning:        true,
		simulatedAnnealing: simulatedAnnealing,
		simulationDuration: simulationDuration,
		snapshotsInterval:  snapshotsInterval,
		simulationStart:    time.Now(),
		lastSnapshot:       time.Now(),
	}
	return g, nil
}

// Update computes a new frame
func (g *Canvas) Update() error {

	if time.Since(g.simulationStart) > g.simulationDuration {
		return SimulationCompleted
	}

	// Intercepts the Space key and starts/stops the execution
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.gameRunning = !g.gameRunning
	}

	if !g.gameRunning {
		return nil
	}

	err := g.savePNG()
	if err != nil {
		panic(err)
	}

	// compute the next simulated annealing iteration
	return g.simulatedAnnealing.Iterate()
}

// Draw writes the computed frame as a byte sequence
func (g *Canvas) Draw(screen *ebiten.Image) {
	screen.WritePixels(g.simulatedAnnealing.ToPixels())
}

// Layout returns the resolution of the canvas
func (g *Canvas) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.width, g.height
}

func (g *Canvas) savePNG() error {
	if !(time.Since(g.lastSnapshot) > g.snapshotsInterval) {
		return nil
	}

	i := g.simulatedAnnealing.GetSnapshot()

	pngFile, err := os.Create(
		fmt.Sprintf("./res/%s_%d-seeds_%d-movreduction_%d.png",
			imageName,
			numSeeds,
			movementReductionFactor,
			int(time.Since(g.simulationStart).Seconds()),
		))
	if err != nil {
		return err
	}

	err = png.Encode(pngFile, i)
	if err != nil {
		return err
	}

	g.lastSnapshot = time.Now()
	return nil
}
