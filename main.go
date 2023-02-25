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
	defaultNumSeeds    = 50
	simulationDuration = 3 * time.Hour
	snapshotsInterval  = 1 * time.Minute
	defaultImageName   = "homer"
)

func main() {

	var numSeeds int
	var inputImageFilePath string

	app := &cli.App{

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "input image",
				Aliases:     []string{"i"},
				Usage:       "path to the input image `FILE`",
				Value:       "./res/" + defaultImageName + ".jpg",
				Destination: &inputImageFilePath,
			},
			&cli.IntFlag{
				Name:        "number of seeds",
				Aliases:     []string{"s"},
				Usage:       "seeds used in the voronoi diagram",
				Value:       defaultNumSeeds,
				Destination: &numSeeds,
			},
		},

		Action: func(c *cli.Context) error {

			targetImage := getTargetImage(inputImageFilePath)

			runSimulatedAnnealing(
				targetImage,
				numSeeds,
			)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func getTargetImage(inputImageFilePath string) TargetImage {

	fileNameWithExt := filepath.Base(inputImageFilePath)
	fileExtension := filepath.Ext(inputImageFilePath)
	fileName := strings.Replace(fileNameWithExt, fileExtension, "", 1)

	reader, openErr := os.Open(inputImageFilePath)
	if openErr != nil {
		panic(openErr)
	}
	defer reader.Close()

	image, _, decodeErr := image.Decode(reader)
	if decodeErr != nil {
		panic(decodeErr)
	}
	bounds := image.Bounds()

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

func runSimulatedAnnealing(
	targetImage TargetImage,
	numSeeds int,
) {

	statFile, err := os.Create(
		fmt.Sprintf("./res/%s_%d-seeds.csv",
			targetImage.Name,
			numSeeds,
		))
	if err != nil {
		panic(err)
	}

	ebiten.SetWindowTitle(
		fmt.Sprintf("Voronoi Simulated Annealing - %ds", numSeeds))
	ebiten.SetWindowSize(targetImage.Width, targetImage.Height)

	voronoi, vErr := NewVoronoi(
		targetImage.Width,
		targetImage.Height,
		numSeeds,
	)
	if vErr != nil {
		panic(vErr)
	}

	simulatedAnnealing, saErr := NewSimulatedAnnealing(
		voronoi,
		targetImage,
		statFile,
	)
	if saErr != nil {
		panic(saErr)
	}

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

	if err := ebiten.RunGame(c); err != nil {
		if err == SimulationCompleted {
			return
		}
		panic(err)
	}
}
