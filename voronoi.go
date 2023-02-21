package main

import (
	"errors"
	"image"
	"image/color"
	"math/rand"
	"time"
)

// Voronoi is the engine used to generate a voronoi diagram on a canvas, starting from auto-generated seed points
type Voronoi struct {

	// diagram size (in pixels)
	width  int
	height int

	// seed configuration of the diagram
	numSeeds int     // number of seeds for the diagram
	seeds    []Point // list of seeds for the diagram

	radius      int     // current radius of the computation
	activeSeeds []Point // list of active seeds to take into account for the computation

	distances               [][]int // precomputed distances matrix (for efficiency reasons)
	r                       *rand.Rand
	movementReductionFactor int

	diagram [][]*Point // resulting diagram (initially empty, to be computed)
}

// NewVoronoi creates a new diagram struct
func NewVoronoi(
	width int,
	height int,
	numSeeds int,
	movementReductionFactor int,
) (*Voronoi, error) {

	if numSeeds > width*height {
		return nil, errors.New("Number of seeds cannot be more than the pixels in the canvas")
	}

	v := Voronoi{
		width:                   width,
		height:                  height,
		numSeeds:                numSeeds,
		seeds:                   []Point{},
		radius:                  0,
		activeSeeds:             []Point{},
		distances:               make([][]int, 3*width+height),
		r:                       rand.New(rand.NewSource(time.Now().UnixNano())),
		movementReductionFactor: movementReductionFactor,
		diagram:                 make([][]*Point, width),
	}
	v.Init()

	return &v, nil
}

// Init initializes the Voronoi diagram and generates a new set of seeds
func (v *Voronoi) Init() {
	v.initDistances()
	v.initDiagram()
	v.initSeeds()
	v.initTessellation()
}

// initDistances populates the precomputed distances matrix,
// to avoid recomputing the same distance values over and over
func (v *Voronoi) initDistances() {

	// the distance vectors needed by the engine can assume values up to twice their dimension  (2*width or 2*height)
	for i := 0; i < 3*v.width+v.height; i++ {

		column := make([]int, 3*v.width+v.height)
		v.distances[i] = column

		for j := 0; j < 3*v.width+v.height; j++ {
			v.distances[i][j] = i*i + j*j
		}
	}
}

// initDiagram populates the diagram with empty points
func (v *Voronoi) initDiagram() {

	for i := 0; i < v.width; i++ {

		column := make([]*Point, v.height)
		v.diagram[i] = column

		for j := 0; j < v.height; j++ {
			v.diagram[i][j] = nil
		}
	}
}

// initSeeds generates a random set of seeds with random colors and stores them in the diagram
func (v *Voronoi) initSeeds() {

	v.seeds = []Point{}

	for i := 0; i < v.numSeeds; i++ {
		x := int(v.r.Intn(v.width))
		y := int(v.r.Intn(v.height))
		d := 0
		seed := Point{
			X:        x,
			Y:        y,
			Distance: &d,
			Color: &color.RGBA{
				R: 0,
				G: 0,
				B: 0,
				A: 255,
			},
		}

		v.seeds = append(v.seeds, seed)
		v.diagram[seed.X][seed.Y] = &seed
	}
}

// initTessellation starts the tessellation of the existing set of seeds
func (v *Voronoi) initTessellation() {

	v.radius = 0
	v.activeSeeds = v.seeds

	// fmt.Println("#######################################")
	// fmt.Println("#### Voronoi tessellation starting ####")
	// fmt.Println("#######################################")
}

/*
Tessellate computes the voronoi diagram

It works on a list of 'active' seeds, where 'active' means that the seed can still extend its area.
At each iteration, the area of the cell corresponding to each seed gets extended by 1 pixel,
and each of these pixels gets assigned to that cell (unless it already belongs to a nearest seed)
*/
func (v *Voronoi) Tessellate() error {

	// the tessellation goes on until all the seeds have extended their area as much as possible
	for len(v.activeSeeds) > 0 {

		stillActiveSeeds := []Point{}
		incrementalVectors := v.getIncrementalVectors()

		// extend the area of each active seed
		for _, seed := range v.activeSeeds {
			// fmt.Println("Iteration starting. Active seeds: ", len(v.activeSeeds))

			// stillActive monitors if the current seed is still able to extend its area
			stillActive := false

			// try to assign the points of the extended area to the current seed
			for _, incrementalVector := range incrementalVectors {
				stillActive = v.assignPointToSeed(
					seed,
					v.distances[abs(incrementalVector.X)][abs(incrementalVector.Y)],
					incrementalVector.X,
					incrementalVector.Y,
				) || stillActive
			}

			// populate the list of the seeds that are still active
			if stillActive {
				stillActiveSeeds = append(stillActiveSeeds, seed)
			}
		}

		v.activeSeeds = stillActiveSeeds
	}

	return nil
}

// assignPointToSeed tries to assign a point to a seed given its relative coordinates
func (v *Voronoi) assignPointToSeed(seed Point, distance int, dx int, dy int) bool {

	// if the point is outside the diagram, ignore it
	if seed.X+dx < 0 ||
		seed.X+dx >= v.width ||
		seed.Y+dy < 0 ||
		seed.Y+dy >= v.height {
		// fmt.Println(fmt.Sprintf("Point (%d,%d) out of canvas, discarded", seed.X+dx, seed.Y+dy))
		return false
	}

	// get the point from the struct containing the resulting diagram representation
	p := v.pointFromDiagram(seed.X+dx, seed.Y+dy)

	// if the point is already assigned to a cell whose seed is closer, ignore it
	if p.Distance != nil && *p.Distance < distance {
		// fmt.Println(fmt.Sprintf("Point (%d,%d) has already a smaller distance (%d < %d), discarded", seed.X+dx, seed.Y+dy, *p.Distance, distance))
		return false
	}

	// the point can be assigned to the seed and stored in the resulting diagram representation
	// fmt.Println(fmt.Sprintf("Assigning point (%d,%d) to cell with seed (%d, %d). Distance: %d", p.X, p.Y, seed.X, seed.Y, distance))
	p.Color = seed.Color
	p.Distance = &distance
	v.diagram[p.X][p.Y] = &p

	return true
}

/*
getIncrementalVectors

It returns a list of points, intended as coordinates relative to the seed,
that represents the new layer of pixels of the expanding cell.

It works by computing a 45° diagonal that has an horizontal (so not orthogonal!)
distance from the seed equal to the radius.
This diagonal is one segment (out of 8) of the diamond surrounding the seed: to compute all
the other segments and get the complete diamond, the algorithm generates all the possible
combinations of the relative coordinates
*/
func (v *Voronoi) getIncrementalVectors() []Point {
	combinations := []Point{}

	v.radius++ // increment the radius of the cell

	// initialize the relative coordinates that will be the first edge of the segment
	dx := v.radius
	dy := 0

	// go on until the other edge of the segment is reached
	for dx >= dy {
		combinations = append(combinations, Point{X: dx, Y: dy})
		combinations = append(combinations, Point{X: dx, Y: -dy})
		combinations = append(combinations, Point{X: -dx, Y: dy})
		combinations = append(combinations, Point{X: -dx, Y: -dy})
		combinations = append(combinations, Point{X: dy, Y: dx})
		combinations = append(combinations, Point{X: dy, Y: -dx})
		combinations = append(combinations, Point{X: -dy, Y: dx})
		combinations = append(combinations, Point{X: -dy, Y: -dx})

		// update the relative coordinates to the next point of the segment
		dx--
		dy++
	}

	return combinations
}

// pointFromDiagram gets the point of the diagram corresponding to the given coordinates
func (v *Voronoi) pointFromDiagram(x int, y int) Point {
	if v.diagram[x][y] == nil {
		v.diagram[x][y] = &Point{
			X: x,
			Y: y,
		}
	}

	return *v.diagram[x][y]
}

func (v *Voronoi) WithSeeds(seeds []Point) {
	v.seeds = seeds
}

func (v *Voronoi) GetSeeds() []Point {
	return v.seeds
}

func (v *Voronoi) Perturbate(temperature float64, seedIndex int) error {

	toPerturbate := v.seeds[seedIndex]
	choice := v.r.Intn(3)
	willPerturbateCoords := choice == 1
	willPerturbateColor := choice == 2
	if choice == 3 {
		willPerturbateCoords = true
		willPerturbateColor = true
	}

	newX := toPerturbate.X
	newY := toPerturbate.Y
	newColor := toPerturbate.Color
	if willPerturbateCoords {
		newX = v.perturbateCoordinate(toPerturbate.X, v.width)
		newY = v.perturbateCoordinate(toPerturbate.Y, v.height)
	} else if willPerturbateColor {
		newColor = &color.RGBA{
			A: 255,
			R: v.perturbateTint(toPerturbate.Color.R, 256),
			G: v.perturbateTint(toPerturbate.Color.G, 256),
			B: v.perturbateTint(toPerturbate.Color.B, 256),
		}
	}

	newSeed := Point{
		X:     newX,
		Y:     newY,
		Color: newColor,
	}

	newSeeds := []Point{}
	newSeeds = append(newSeeds, v.seeds...)
	newSeeds[seedIndex] = newSeed
	v.seeds = newSeeds

	v.initDiagram()
	v.initTessellation()
	return nil
}

func (v *Voronoi) perturbateCoordinate(currentCoordinate int, maxValue int) int {
	var newCoordinate int

	movement := v.r.Float64() * float64(maxValue) / float64(v.movementReductionFactor)
	multiplier := float64(v.r.Intn(2)*2 - 1)
	newCoordinate = currentCoordinate + int(multiplier*movement)

	if newCoordinate >= maxValue {
		newCoordinate = maxValue - 1
	} else if newCoordinate < 0 {
		newCoordinate = 0
	}

	return newCoordinate
}

func (v *Voronoi) perturbateTint(currentTint byte, maxValue int) uint8 {
	var newTint int

	movement := v.r.Float64() * float64(maxValue)
	multiplier := v.r.Intn(2)*2 - 1

	newTint = int(currentTint) + int(float64(multiplier)*movement)

	if newTint >= maxValue {
		newTint = maxValue - 1
	} else if newTint < 0 {
		newTint = 0
	}
	return uint8(newTint)
}

// ToPixels generates the byte array containing the information to render the diagram.
// Each row of the canvas is concatenated to obtain a one-dimensional array.
// Each pixel is represented by 4 bytes, representing the Red, Green, Blue and Alpha info.
func (v *Voronoi) ToPixels() []byte {

	pixels := make([]byte, v.width*v.height*4)

	// iterate through each pixel
	for i := 0; i < v.width; i++ {
		for j := 0; j < v.height; j++ {
			pos := (j*v.width + i) * 4

			if v.diagram[i][j] != nil && v.diagram[i][j].Color != nil {
				pixels[pos] = v.diagram[i][j].Color.R
				pixels[pos+1] = v.diagram[i][j].Color.G
				pixels[pos+2] = v.diagram[i][j].Color.B
				pixels[pos+3] = v.diagram[i][j].Color.A

			} else {
				// if the point has not assigned any color yet, show it as black
				pixels[pos] = 0
				pixels[pos+1] = 0
				pixels[pos+2] = 0
				pixels[pos+3] = 0
			}
		}
	}

	// iterate through the seeds to render them as black points
	for _, s := range v.seeds {
		pos := (s.Y*v.width + s.X) * 4
		pixels[pos] = 0
		pixels[pos+1] = 0
		pixels[pos+2] = 0
		pixels[pos+3] = 0
	}

	return pixels
}

func (v *Voronoi) ToImage() image.Image {
	res := image.NewRGBA(image.Rect(0, 0, v.width, v.height))

	// iterate through each pixel
	for i := 0; i < v.width; i++ {
		for j := 0; j < v.height; j++ {

			c := color.RGBA{
				R: 0,
				G: 0,
				B: 0,
				A: 255,
			}
			if v.diagram[i][j] != nil && v.diagram[i][j].Color != nil {
				c = *v.diagram[i][j].Color
			}
			res.Set(i, j, c)
		}
	}

	return res
}
