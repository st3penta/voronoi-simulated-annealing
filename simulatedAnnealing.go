package main

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"time"
)

// SimulatedAnnealing is the engine driving the annealing.
// At each iteration, it creates a new perturbation of the current solution, and evaluates its temperature.
// The temperature of a solution is the distance of the solution from the target,
// and the engine tries to reduce it by trial and error
type SimulatedAnnealing struct {
	voronoi         VoronoiDiagram // voronoi engine used to generate the images used for each annealing iteration
	targetImage     TargetImage    // image to be used as target for the annealing algorithm
	startingTime    time.Time      // time mark of the beginning of the simulation
	statFile        *os.File       // csv file logging the temperature in function of time, for further analysis
	r               *rand.Rand     // generator for random numbers used in the computations
	temperature     float64        // temperature of the current solution of the annealing. It can assume values in the interval [0,1]
	maxHeat         float64        // max temperature of the image (needed for normalization purposes)
	bestTemperature float64        // tracker of the best temperature reached by the algorithm
	bestSolution    []Point        // tracker of the solution associated with the best temperature. The algorithm is reset to this state when the temperature grows out of control
}

// NewSimulatedAnnealing initializes the simulated annealing engine
func NewSimulatedAnnealing(
	voronoi VoronoiDiagram,
	targetImage TargetImage,
	statFile *os.File,
) (*SimulatedAnnealing, error) {

	// initialize the csv file to track the progress of the algorithm
	_, err := statFile.WriteString("elapsed_seconds,temperature\n")
	if err != nil {
		return nil, err
	}

	//compute the maximum head of the image, as number of pixels in the image times the max RGBA distance for each pixel
	maxHeat := float64(4 * 255 * targetImage.Width * targetImage.Height)

	return &SimulatedAnnealing{
		voronoi:         voronoi,
		targetImage:     targetImage,
		maxHeat:         maxHeat,
		bestTemperature: 1.0,
		bestSolution:    nil,
		temperature:     1.0,
		startingTime:    time.Now(),
		statFile:        statFile,
		r:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Iterate is the core function of the engine.
//
// At each iteration, the engine perturbates the current solution and evaluates its temperature.
// The new temperature is automatically accepted if is lower than the previous one, but it may also
// be accepted if is higher, based on a probabilistic function.
//
// Since regressions are possible, even if the probability is low the temperature could grow indefinitely,
// so a reset mechanism is put in place to reset the state of the annealing if it grows too much out of control
func (sa *SimulatedAnnealing) Iterate() error {

	// keep a copy of the current solution, so the system can be
	// resetted to this state if the perturbation is not acceptable
	currentSeeds := sa.voronoi.GetSeeds()

	// compute the number of perturbations in function of the temperature.
	// the higher the temperature, the more perturbations are performed:
	// in this way, at highest temperatures furthest perturbations are evaluated,
	// increasing the ability to explore the solution space.
	//
	// At max temperature (t = 1.0), the number of perturbations corresponds to a third of the seeds,
	// and this number gets lower as the temperature lowers
	perturbations := int(math.Floor(sa.temperature * float64(len(currentSeeds)) / 3))
	if perturbations == 0 {
		perturbations = 1
	}

	// Perturbate the current solution as many times as computed in the previous step.
	for j := 0; j < perturbations; j++ {
		pErr := sa.voronoi.Perturbate(
			sa.temperature,
		)
		if pErr != nil {
			return pErr
		}
	}

	// compute the voronoi diagram solution given the perturbated seeds
	vErr := sa.voronoi.Tessellate()
	if vErr != nil {
		return vErr
	}

	// compute the temperature of the perturbated solution
	newTemperature := sa.computeTemperature()

	// evaluate the new temperature
	if !sa.isAcceptableTemperature(newTemperature) {
		// if the new temperature is not accepted, reset the algorightm to its previous state
		sa.voronoi.WithSeeds(currentSeeds)
		return nil
	}

	// check if the new temperature is running out of control, and if so reset it to the best solution so far
	if (newTemperature - sa.bestTemperature) > sa.bestTemperature/10 {
		fmt.Printf("Current temperature exceeded 10 percent threshold, restarting from the best solution so far: %.10f\n", sa.bestTemperature)

		sa.voronoi.WithSeeds(sa.bestSolution)
		sa.temperature = sa.bestTemperature
		return nil
	}

	// update the simulated annealing state, and log the iteration
	sa.temperature = newTemperature
	err := sa.logIteration()
	if err != nil {
		return err
	}

	// update the best temperature hook
	if sa.temperature < sa.bestTemperature {
		sa.bestTemperature = sa.temperature
		sa.bestSolution = sa.voronoi.GetSeeds()
	}

	return nil
}

// computeTemperature computes the temperature of the current solution, intended as
// the distance of the RGBA values of each pixel from the corresponding pixel of the target image
func (sa *SimulatedAnnealing) computeTemperature() float64 {

	// get the pixels of the current solution
	currentSolution := sa.voronoi.ToPixels()
	heat := 0.0 // keep track of the total heat of the current solution

	// iterate each RGBA value of each pixel in the target image
	for i, b := range sa.targetImage.Bytes {

		// compute the current and target values of the current color component
		targetValue := int(b)
		currentValue := int(currentSolution[i])

		// add to the total heat the distance between the current value and the target value
		heat += math.Abs(float64(targetValue - currentValue))
	}

	// return the normalized heat (aka temperature)
	return heat / sa.maxHeat
}

// isAcceptableTemperature decides if the input temperature can be accepted compared
// to the temperature of the previous state
func (sa *SimulatedAnnealing) isAcceptableTemperature(temperature float64) bool {

	// if the new temperature is lower than the previous one, always accept it.
	// if we want to use a hill climbing approach, this check is all that we need
	if temperature <= sa.temperature {
		return true
	}

	// if the new temperature is higher than the previous one, accept it using a probabilistic approach.
	// First, a random number in the interval [0, 1] is generated.
	// Then, the difference in temperatures (normalized in the [0, 1] interval) is calculated.
	// The random number is compared to the normalized temperature, and if it's greater, the temperature is accepted.
	//
	// But there's a plot twist! The normalized temperature difference is passed to a
	// sigmoid function (https://en.wikipedia.org/wiki/Sigmoid_function), that enhances the
	// probability of accepting lower differences
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

// ToPixels returns the pixels of the current solution
func (sa *SimulatedAnnealing) ToPixels() []byte {
	return sa.voronoi.ToPixels()
}

// GetSnapshot returns the image representation of the current solution
func (sa *SimulatedAnnealing) GetSnapshot() image.Image {
	return sa.voronoi.ToImage()
}
