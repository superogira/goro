package scene

import (
	"math"
)

type Point3 struct {
	X float64
	Y float64
	Z float64
}

type Mat4 [16]float64

func Sub3(a, b Point3) Point3 {
	return Point3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func Add3(a, b Point3) Point3 {
	return Point3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

func Mul3(v Point3, scalar float64) Point3 {
	return Point3{X: v.X * scalar, Y: v.Y * scalar, Z: v.Z * scalar}
}

func Cross3(a, b Point3) Point3 {
	return Point3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func Normalize3(v Point3) Point3 {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if length == 0 {
		return Point3{}
	}
	return Point3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}
}

func DegreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}
