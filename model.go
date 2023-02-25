package main

import "image/color"

// TargetImage is the struct containing info about the target image: its name, size, and the RGBA values of its pixels
type TargetImage struct {
	Name   string
	Bytes  []byte
	Width  int
	Height int
}

// Point is the struct modeling a point of the Voronoi diagram, with its position, color, and distance from the center of the seed
type Point struct {
	X        int
	Y        int
	Distance *int
	Color    *color.RGBA
}

// abs is a utility function to compute the absolute value of an int
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
