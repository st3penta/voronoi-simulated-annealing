package main

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"time"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// SimulationCompleted is the error returned when the simulation ends by timeout
var SimulationCompleted = errors.New("Simulation completed")

// Canvas handles the canvas visualization
type Canvas struct {
	imageName string
	numSeeds  int

	// resolution of the canvas
	width  int
	height int

	gameRunning bool

	// simulated annealing info
	simulatedAnnealing SimulatedAnnealingEngine
	simulationDuration time.Duration
	simulationStart    time.Time

	// snapshots logic timers
	lastSnapshot      time.Time
	snapshotsInterval time.Duration
}

// NewCanvas creates a canvas with the simulated annealing ready to start
func NewCanvas(
	imageName string,
	numSeeds int,
	width int,
	height int,
	simulatedAnnealing SimulatedAnnealingEngine,
	simulationDuration time.Duration,
	snapshotsInterval time.Duration,
) (*Canvas, error) {

	g := &Canvas{
		imageName:          imageName,
		numSeeds:           numSeeds,
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

	// end the simulation if the timeout has been reached
	if time.Since(g.simulationStart) > g.simulationDuration {
		return SimulationCompleted
	}

	// intercept the Space key and start/stop the execution
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.gameRunning = !g.gameRunning
	}
	if !g.gameRunning {
		return nil
	}

	// take periodic snapshots of the canvas
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

// savePNG saves periodic snapshots of the canvas
func (g *Canvas) savePNG() error {

	// skip the saving if the last snapshot is still too recent
	if !(time.Since(g.lastSnapshot) > g.snapshotsInterval) {
		return nil
	}

	// get the image data
	i := g.simulatedAnnealing.GetSnapshot()

	// create a png file for the snapshot
	pngFile, err := os.Create(
		fmt.Sprintf("./res/%s_%d-seeds_%d.png",
			g.imageName,
			g.numSeeds,
			int(time.Since(g.simulationStart).Seconds()),
		))
	if err != nil {
		return err
	}

	// save the data into the png file
	err = png.Encode(pngFile, i)
	if err != nil {
		return err
	}

	// reset the snapshot interval timer
	g.lastSnapshot = time.Now()

	return nil
}
