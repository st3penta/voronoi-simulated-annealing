package main

type TargetImage struct {
	Bytes  []byte
	Width  int
	Height int
}

type Color struct {
	R byte
	G byte
	B byte
	A byte
}

type Point struct {
	X        int
	Y        int
	Distance *int
	Color    *Color
}

// abs is a utility function to compute the absolute value of an int
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
