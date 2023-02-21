package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"time"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	"github.com/urfave/cli/v2"
)

var (
	numSeeds                = 50
	movementReductionFactor = 10
	percentThreshold        = 10
	simulationDuration      = 3 * time.Hour
	snapshotsInterval       = 1 * time.Minute

	imageName = "homer"
)

func main() {
	var inputImageFilePath string

	app := &cli.App{

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "input image",
				Aliases:     []string{"i"},
				Usage:       "path to the input image `FILE`",
				Value:       "./res/" + imageName + ".jpg",
				Destination: &inputImageFilePath,
			},
		},

		Action: func(cCtx *cli.Context) error {

			targetImage := getTargetImage(inputImageFilePath)

			runSimulatedAnnealing(targetImage)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}

func getTargetImage(inputImageFilePath string) TargetImage {

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
		Bytes:  imageBytes,
		Width:  bounds.Max.X - bounds.Min.X,
		Height: bounds.Max.Y - bounds.Min.Y,
	}
}

func runSimulatedAnnealing(
	targetImage TargetImage,
) {

	statFile, err := os.Create(
		fmt.Sprintf("./res/%s_%d-seeds_%d-movreduction.csv",
			imageName,
			numSeeds,
			movementReductionFactor,
		))
	if err != nil {
		panic(err)
	}

	ebiten.SetWindowTitle(
		fmt.Sprintf("Voronoi Simulated Annealing - %ds, %dm",
			numSeeds,
			movementReductionFactor,
		))
	ebiten.SetWindowSize(targetImage.Width, targetImage.Height)

	voronoi, vErr := NewVoronoi(
		targetImage.Width,
		targetImage.Height,
		numSeeds,
		movementReductionFactor,
	)
	if vErr != nil {
		panic(vErr)
	}

	simulatedAnnealing, saErr := NewSimulatedAnnealing(
		voronoi,
		targetImage,
		statFile,
		percentThreshold,
	)
	if saErr != nil {
		panic(saErr)
	}

	c, cErr := NewCanvas(
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
