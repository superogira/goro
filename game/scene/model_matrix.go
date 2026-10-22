package scene

import (
	"math"
)

func Mat4TransformVector(matrix Mat4, point Point3) Point3 {
	return Point3{
		X: matrix[0]*point.X + matrix[4]*point.Y + matrix[8]*point.Z,
		Y: matrix[1]*point.X + matrix[5]*point.Y + matrix[9]*point.Z,
		Z: matrix[2]*point.X + matrix[6]*point.Y + matrix[10]*point.Z,
	}
}

func Mat4Translate(matrix Mat4, v Point3) Mat4 {
	t := Mat4Identity()
	t[12] = v.X
	t[13] = v.Y
	t[14] = v.Z
	return Mat4Multiply(matrix, t)
}

func Mat4Scale(matrix Mat4, v Point3) Mat4 {
	s := Mat4Identity()
	s[0] = v.X
	s[5] = v.Y
	s[10] = v.Z
	return Mat4Multiply(matrix, s)
}

func Mat4RotateX(matrix Mat4, angle float64) Mat4 {
	s, c := math.Sin(angle), math.Cos(angle)
	r := Mat4Identity()
	r[5] = c
	r[6] = s
	r[9] = -s
	r[10] = c
	return Mat4Multiply(matrix, r)
}

func Mat4RotateY(matrix Mat4, angle float64) Mat4 {
	s, c := math.Sin(angle), math.Cos(angle)
	r := Mat4Identity()
	r[0] = c
	r[2] = -s
	r[8] = s
	r[10] = c
	return Mat4Multiply(matrix, r)
}

func Mat4RotateZ(matrix Mat4, angle float64) Mat4 {
	s, c := math.Sin(angle), math.Cos(angle)
	r := Mat4Identity()
	r[0] = c
	r[1] = s
	r[4] = -s
	r[5] = c
	return Mat4Multiply(matrix, r)
}

func Mat4RotateAxis(matrix Mat4, axis Point3, angle float64) Mat4 {
	axis = Normalize3(axis)
	if axis == (Point3{}) || angle == 0 {
		return matrix
	}
	x, y, z := axis.X, axis.Y, axis.Z
	s, c := math.Sin(angle), math.Cos(angle)
	t := 1 - c
	r := Mat4Identity()
	r[0] = x*x*t + c
	r[1] = y*x*t + z*s
	r[2] = z*x*t - y*s
	r[4] = x*y*t - z*s
	r[5] = y*y*t + c
	r[6] = z*y*t + x*s
	r[8] = x*z*t + y*s
	r[9] = y*z*t - x*s
	r[10] = z*z*t + c
	return Mat4Multiply(matrix, r)
}

func Mat4RotateQuat(matrix Mat4, q [4]float32) Mat4 {
	x, y, z, w := float64(q[0]), float64(q[1]), float64(q[2]), float64(q[3])
	length := math.Sqrt(x*x + y*y + z*z + w*w)
	if length == 0 {
		return matrix
	}
	x /= length
	y /= length
	z /= length
	w /= length

	x2, y2, z2 := x+x, y+y, z+z
	xx, xy, xz := x*x2, x*y2, x*z2
	yy, yz, zz := y*y2, y*z2, z*z2
	wx, wy, wz := w*x2, w*y2, w*z2

	r := Mat4Identity()
	r[0] = 1 - yy - zz
	r[1] = xy + wz
	r[2] = xz - wy
	r[4] = xy - wz
	r[5] = 1 - xx - zz
	r[6] = yz + wx
	r[8] = xz + wy
	r[9] = yz - wx
	r[10] = 1 - xx - yy
	return Mat4Multiply(matrix, r)
}

func Mat4FromMat3(matrix [9]float32) Mat4 {
	out := Mat4Identity()
	out[0] = float64(matrix[0])
	out[1] = float64(matrix[1])
	out[2] = float64(matrix[2])
	out[4] = float64(matrix[3])
	out[5] = float64(matrix[4])
	out[6] = float64(matrix[5])
	out[8] = float64(matrix[6])
	out[9] = float64(matrix[7])
	out[10] = float64(matrix[8])
	return out
}

func Mat4TransformPoint(matrix Mat4, point Point3) Point3 {
	return Point3{
		X: matrix[0]*point.X + matrix[4]*point.Y + matrix[8]*point.Z + matrix[12],
		Y: matrix[1]*point.X + matrix[5]*point.Y + matrix[9]*point.Z + matrix[13],
		Z: matrix[2]*point.X + matrix[6]*point.Y + matrix[10]*point.Z + matrix[14],
	}
}
