package main

import (
	"fmt"
	"math"
	"math/rand"
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
	startingTime     time.Time
	statFile         *os.File
	seedReiterations int
	r                *rand.Rand
	bestSolution     []Point
	bestTemperature  int
	percentThreshold int
}

func NewSimulatedAnnealing(
	voronoi VoronoiDiagram,
	targetImage TargetImage,
	statFile *os.File,
	seedReiterations int,
	percentThreshold int,
) (*SimulatedAnnealing, error) {

	_, err := statFile.WriteString("elapsed_seconds,temperature\n")
	if err != nil {
		return nil, err
	}

	maxTemp := 4 * 256 * targetImage.Width * targetImage.Height
	return &SimulatedAnnealing{
		voronoi:          voronoi,
		targetImage:      targetImage,
		temperature:      maxTemp,
		startingTime:     time.Now(),
		statFile:         statFile,
		seedReiterations: seedReiterations,
		r:                rand.New(rand.NewSource(time.Now().UnixNano())),
		bestSolution:     nil,
		bestTemperature:  maxTemp,
		percentThreshold: percentThreshold,
	}, nil
}

func (sa *SimulatedAnnealing) Iterate() error {
	currentSeeds := sa.voronoi.GetSeeds()

	for i := 0; i < sa.seedReiterations; i++ {

		pErr := sa.voronoi.Perturbate(
			sa.temperature,
			sa.r.Intn(len(currentSeeds)),
		)
		if pErr != nil {
			return pErr
		}

		vErr := sa.voronoi.Tessellate()
		if vErr != nil {
			return vErr
		}

		newTemperature := sa.computeTemperature()

		if !sa.isAcceptableTemperature(newTemperature) {
			break
		}
		sa.temperature = newTemperature

		if sa.temperature < sa.bestTemperature {
			sa.bestTemperature = sa.temperature
			sa.bestSolution = sa.voronoi.GetSeeds()
		}

		err := sa.logIteration()
		if err != nil {
			return err
		}

		if (newTemperature - sa.bestTemperature) > sa.bestTemperature*sa.percentThreshold/100 {
			fmt.Printf("Current temperature exceeded %d percent threshold, restarting from the best solution so far: %d\n", sa.percentThreshold, sa.bestTemperature)

			sa.voronoi.WithSeeds(sa.bestSolution)
			sa.temperature = sa.bestTemperature
			break
		}

		sa.voronoi.WithSeeds(currentSeeds)
	}

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
	if temperature < sa.temperature {
		return true
	}

	prob := 1 / math.Log10(float64(temperature-sa.temperature))
	rand := sa.r.Float64()
	return prob > rand
}

func (sa *SimulatedAnnealing) logIteration() error {
	fmt.Printf(
		"Current temperature: %d, time passed: %s\n",
		sa.temperature,
		time.Since(sa.startingTime),
	)

	_, err := sa.statFile.WriteString(
		fmt.Sprintf("%.0f,%d\n",
			time.Since(sa.startingTime).Seconds(),
			sa.temperature),
	)
	return err
}

func (sa *SimulatedAnnealing) ToPixels() []byte {
	return sa.voronoi.ToPixels()
}
