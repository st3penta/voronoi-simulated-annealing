package main

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"time"
)

// VoronoiDiagram is the voronoi engine
type VoronoiDiagram interface {
	Init()
	Tessellate() error
	Perturbate(temperature float64, seedIndex int) error
	ToPixels() []byte
	ToImage() image.Image
	GetSeeds() []Point
	WithSeeds([]Point)
}

type SimulatedAnnealing struct {
	voronoi          VoronoiDiagram
	targetImage      TargetImage
	startingTime     time.Time
	statFile         *os.File
	r                *rand.Rand
	temperature      float64
	maxHeat          float64
	bestTemperature  float64
	bestSolution     []Point
	percentThreshold int
}

func NewSimulatedAnnealing(
	voronoi VoronoiDiagram,
	targetImage TargetImage,
	statFile *os.File,
	percentThreshold int,
) (*SimulatedAnnealing, error) {

	_, err := statFile.WriteString("elapsed_seconds,temperature\n")
	if err != nil {
		return nil, err
	}

	maxHeat := float64(4 * 255 * targetImage.Width * targetImage.Height)

	return &SimulatedAnnealing{
		voronoi:          voronoi,
		targetImage:      targetImage,
		maxHeat:          maxHeat,
		bestTemperature:  1.0,
		bestSolution:     nil,
		temperature:      1.0,
		startingTime:     time.Now(),
		statFile:         statFile,
		r:                rand.New(rand.NewSource(time.Now().UnixNano())),
		percentThreshold: percentThreshold,
	}, nil
}

func (sa *SimulatedAnnealing) Iterate() error {
	currentSeeds := sa.voronoi.GetSeeds()

	perturbations := int(math.Floor(sa.temperature * float64(len(currentSeeds)) / 3))
	if perturbations == 0 {
		perturbations = 1
	}

	for j := 0; j < perturbations; j++ {
		pErr := sa.voronoi.Perturbate(
			sa.temperature,
			sa.r.Intn(len(currentSeeds)),
		)
		if pErr != nil {
			return pErr
		}
	}

	vErr := sa.voronoi.Tessellate()
	if vErr != nil {
		return vErr
	}

	newTemperature := sa.computeTemperature()

	if !sa.isAcceptableTemperature(newTemperature) {
		sa.voronoi.WithSeeds(currentSeeds)
		return nil
	}

	if (newTemperature - sa.bestTemperature) > sa.bestTemperature*float64(sa.percentThreshold)/100 {
		fmt.Printf("Current temperature exceeded %d percent threshold, restarting from the best solution so far: %.10f\n", sa.percentThreshold, sa.bestTemperature)

		sa.voronoi.WithSeeds(sa.bestSolution)
		sa.temperature = sa.bestTemperature
		return nil
	}

	sa.temperature = newTemperature
	err := sa.logIteration()
	if err != nil {
		return err
	}

	if sa.temperature < sa.bestTemperature {
		sa.bestTemperature = sa.temperature
		sa.bestSolution = sa.voronoi.GetSeeds()
	}

	return nil
}

func (sa *SimulatedAnnealing) computeTemperature() float64 {

	currentSolution := sa.voronoi.ToPixels()
	heat := 0.0
	for i, b := range sa.targetImage.Bytes {

		targetValue := int(b)
		currentValue := int(currentSolution[i])
		heat += math.Abs(float64(targetValue - currentValue))
	}

	return heat / sa.maxHeat
}

func (sa *SimulatedAnnealing) isAcceptableTemperature(temperature float64) bool {
	if temperature <= sa.temperature {
		return true
	}

	rand := sa.r.Float64()
	percDiff := (temperature - sa.temperature) * 100 / sa.temperature
	sigmoid := (2 / (1 + math.Exp(-10*percDiff))) - 1 // sigmoid function variation
	return rand > sigmoid
}

func (sa *SimulatedAnnealing) logIteration() error {
	fmt.Printf(
		"Current temperature: %.10f, time passed: %s\n",
		sa.temperature,
		time.Since(sa.startingTime),
	)

	_, err := sa.statFile.WriteString(
		fmt.Sprintf("%.0f,%.10f\n",
			time.Since(sa.startingTime).Seconds(),
			sa.temperature),
	)
	return err
}

func (sa *SimulatedAnnealing) ToPixels() []byte {
	return sa.voronoi.ToPixels()
}

func (sa *SimulatedAnnealing) GetSnapshot() image.Image {
	return sa.voronoi.ToImage()
}
