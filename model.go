package main

import "image/color"

type TargetImage struct {
	Name   string
	Bytes  []byte
	Width  int
	Height int
}

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
