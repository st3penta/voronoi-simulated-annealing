package main

import (
	ebiten "github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type SimulatedAnnealingEngine interface {
	Iterate() error
	ToPixels() []byte
}

// Canvas handles the canvas visualization
type Canvas struct {

	// resolution of the canvas
	width  int
	height int

	gameRunning bool

	simulatedAnnealing SimulatedAnnealingEngine
}

// NewCanvas creates a canvas with the simulated annealing ready to start
func NewCanvas(
	width int,
	height int,
	simulatedAnnealing SimulatedAnnealingEngine,
) (*Canvas, error) {

	g := &Canvas{
		width:              width,
		height:             height,
		gameRunning:        true,
		simulatedAnnealing: simulatedAnnealing,
	}
	return g, nil
}

// Update computes a new frame
func (g *Canvas) Update() error {

	// Intercepts the Space key and starts/stops the execution
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.gameRunning = !g.gameRunning
	}

	if g.gameRunning {
		// compute the next simulated annealing iteration
		return g.simulatedAnnealing.Iterate()
	}
	return nil
}

// Draw writes the computed frame as a byte sequence
func (g *Canvas) Draw(screen *ebiten.Image) {
	screen.WritePixels(g.simulatedAnnealing.ToPixels())
}

// Layout returns the resolution of the canvas
func (g *Canvas) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.width, g.height
}
