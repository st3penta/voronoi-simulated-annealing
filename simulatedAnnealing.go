package main

import (
	"fmt"
	"os"
	"time"
)

// VoronoiDiagram is the voronoi engine
type VoronoiDiagram interface {
	Init()
	Tessellate() error
	Perturbate(temperature int, seedIndex int) error
	ToPixels() []byte
	GetSeeds() []Point
	WithSeeds([]Point)
}

type SimulatedAnnealing struct {
	voronoi          VoronoiDiagram
	targetImage      TargetImage
	temperature      int
	currentSeed      int
	startingTime     time.Time
	statFile         *os.File
	seedReiterations int
}

func NewSimulatedAnnealing(
	voronoi VoronoiDiagram,
	targetImage TargetImage,
	statFile *os.File,
	seedReiterations int,
) (*SimulatedAnnealing, error) {

	_, err := statFile.WriteString("temperature,elapsed_seconds\n")
	if err != nil {
		return nil, err
	}

	return &SimulatedAnnealing{
		voronoi:          voronoi,
		targetImage:      targetImage,
		temperature:      4 * 256 * targetImage.Width * targetImage.Height,
		currentSeed:      0,
		startingTime:     time.Now(),
		statFile:         statFile,
		seedReiterations: seedReiterations,
	}, nil
}

func (sa *SimulatedAnnealing) Iterate() error {
	currentSeeds := sa.voronoi.GetSeeds()

	for i := 0; i < 3; i++ {

		pErr := sa.voronoi.Perturbate(sa.temperature, sa.currentSeed)
		if pErr != nil {
			return pErr
		}

		vErr := sa.voronoi.Tessellate()
		if vErr != nil {
			return vErr
		}

		newTemperature := sa.computeTemperature()

		if sa.isAcceptableTemperature(newTemperature) {
			sa.temperature = newTemperature

			err := sa.logIteration()
			if err != nil {
				return err
			}

			break
		} else {
			sa.voronoi.WithSeeds(currentSeeds)
		}
	}
	sa.currentSeed++
	sa.currentSeed = sa.currentSeed % len(currentSeeds)

	return nil
}

func (sa *SimulatedAnnealing) computeTemperature() int {

	currentSolution := sa.voronoi.ToPixels()
	temperature := 0
	for i, b := range sa.targetImage.Bytes {
		targetValue := int(b)
		currentValue := int(currentSolution[i])
		temperature += abs(targetValue - currentValue)
	}

	return temperature
}

func (sa *SimulatedAnnealing) isAcceptableTemperature(temperature int) bool {
	return temperature < sa.temperature
}

func (sa *SimulatedAnnealing) logIteration() error {
	fmt.Println(
		fmt.Sprintf("Current temperature: %d, time passed: %s",
			sa.temperature,
			time.Since(sa.startingTime)),
	)

	_, err := sa.statFile.WriteString(
		fmt.Sprintf("%d,%.0f\n",
			sa.temperature,
			time.Since(sa.startingTime).Seconds()),
	)
	return err
}

func (sa *SimulatedAnnealing) ToPixels() []byte {
	return sa.voronoi.ToPixels()
}
