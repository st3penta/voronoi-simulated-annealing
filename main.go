package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"time"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	"github.com/urfave/cli/v2"
)

var (
	// defaults argument values for the `run` command
	defaultNumSeeds           = 50
	defaultSimulationDuration = 3 * time.Hour
	defaultSnapshotsInterval  = 1 * time.Minute
	defaultImageName          = "homer"
)

func main() {

	//
	// CLI initialization
	//
	var numSeeds int
	var inputImageFilePath string
	var simulationDuration time.Duration
	var snapshotsInterval time.Duration

	app := &cli.App{

		Name: "Voronoi Simulated Annealing",

		Usage: "Simulated Annealing approximation using Voronoi diagrams",

		Description: "This simulation takes an image in input and tries to approximate it with a voronoi diagram, by using the simulate annealing approach",

		UsageText: "voronoi-simulated-annealing [command] [options]",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "targetImage",
				Aliases:     []string{"i"},
				Usage:       "Path to the input image `FILE` to be used as target image for the annealing. Only JPG images are supported",
				Value:       "./res/" + defaultImageName + ".jpg",
				Destination: &inputImageFilePath,
			},
			&cli.IntFlag{
				Name:        "seedsNumber",
				Aliases:     []string{"n"},
				Usage:       "Number of seeds (cells) used in the voronoi diagram",
				Value:       defaultNumSeeds,
				Destination: &numSeeds,
			},
			&cli.DurationFlag{
				Name:        "simulationDuration",
				Aliases:     []string{"d"},
				Usage:       "Duration of the simulation",
				Value:       defaultSimulationDuration,
				Destination: &simulationDuration,
			},
			&cli.DurationFlag{
				Name:        "snapshotsInterval",
				Aliases:     []string{"s"},
				Usage:       "Time interval between the snapshots taken during the simulation (to track the progresses)",
				Value:       defaultSnapshotsInterval,
				Destination: &snapshotsInterval,
			},
		},

		Commands: []*cli.Command{
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "Runs the simulated annealing",
				Action: func(cCtx *cli.Context) error {
					targetImage := getTargetImage(inputImageFilePath)

					runSimulatedAnnealing(
						targetImage,
						numSeeds,
						simulationDuration,
						snapshotsInterval,
					)
					return nil
				},
			},
		},
	}

	// run the cli
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

// getTargetImage reads the target image at the specified path, and extracts the RGB values of each pixel
func getTargetImage(inputImageFilePath string) TargetImage {

	// get file name stripped from path and extension
	fileNameWithExt := filepath.Base(inputImageFilePath)
	fileExtension := filepath.Ext(inputImageFilePath)
	fileName := strings.Replace(fileNameWithExt, fileExtension, "", 1)

	// open the image file
	reader, openErr := os.Open(inputImageFilePath)
	if openErr != nil {
		panic(openErr)
	}
	defer reader.Close()

	// decode the image using the JPG decoder
	image, _, decodeErr := image.Decode(reader)
	if decodeErr != nil {
		panic(decodeErr)
	}
	bounds := image.Bounds()

	// extract the 8-bit RGBA values of the pixels
	imageBytes := []byte{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := image.At(x, y).RGBA()
			imageBytes = append(
				imageBytes,
				byte(r/256),
				byte(g/256),
				byte(b/256),
				byte(a/256),
			)
		}
	}

	return TargetImage{
		Name:   fileName,
		Bytes:  imageBytes,
		Width:  bounds.Max.X - bounds.Min.X,
		Height: bounds.Max.Y - bounds.Min.Y,
	}
}

// runSimulatedAnnealing initializes the structs needed to run the simulation, and starts it
func runSimulatedAnnealing(
	targetImage TargetImage,
	numSeeds int,
	simulationDuration time.Duration,
	snapshotsInterval time.Duration,
) {

	// create a file that logs the temperature progresses in function of time since the start of the simulation, for further analysis
	statFile, err := os.Create(
		fmt.Sprintf("./res/%s_%d-seeds.csv",
			targetImage.Name,
			numSeeds,
		))
	if err != nil {
		panic(err)
	}

	// initialize the Voronoi diagram
	voronoi, vErr := NewVoronoi(
		targetImage.Width,
		targetImage.Height,
		numSeeds,
	)
	if vErr != nil {
		panic(vErr)
	}

	// initialize the simulated annealing
	simulatedAnnealing, saErr := NewSimulatedAnnealing(
		voronoi,
		targetImage,
		statFile,
	)
	if saErr != nil {
		panic(saErr)
	}

	// initialize the canvas for the GUI
	c, cErr := NewCanvas(
		targetImage.Name,
		numSeeds,
		targetImage.Width,
		targetImage.Height,
		simulatedAnnealing,
		simulationDuration,
		snapshotsInterval,
	)
	if cErr != nil {
		panic(cErr)
	}

	// initialize the system window size and title
	ebiten.SetWindowTitle(
		fmt.Sprintf("Voronoi Simulated Annealing (%d seeds)", numSeeds))
	ebiten.SetWindowSize(targetImage.Width, targetImage.Height)

	// run the simulation
	if err := ebiten.RunGame(c); err != nil {
		if err == SimulationCompleted {
			return
		}
		panic(err)
	}
}
