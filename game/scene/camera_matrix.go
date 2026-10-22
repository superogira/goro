package scene

import "math"

func Mat4Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func Mat4Multiply(a, b Mat4) Mat4 {
	var out Mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			out[col*4+row] =
				a[0*4+row]*b[col*4+0] +
					a[1*4+row]*b[col*4+1] +
					a[2*4+row]*b[col*4+2] +
					a[3*4+row]*b[col*4+3]
		}
	}
	return out
}

func NewProjection(width, height int, targetX, targetY, targetZ, yaw, pitch, zoom float64) Projection {
	projection := Projection{
		PlayerX:     targetX,
		PlayerY:     targetY,
		PlayerZ:     targetZ,
		ScreenW:     float64(width),
		ScreenH:     float64(height),
		CameraYaw:   yaw,
		CameraPitch: ClampCameraPitch(pitch),
		CameraZoom:  NormalizeSceneCameraZoom(zoom),
	}
	projection.ViewProjection = CameraMatrix(float64(width), float64(height), targetX, targetY, targetZ, yaw, projection.CameraPitch, projection.CameraZoom)
	return projection
}

func CameraMatrix(width, height, targetX, targetY, targetZ, yawDegrees, pitchDegrees, zoom float64) Mat4 {
	distance := NormalizeSceneCameraZoom(zoom) * 0.5
	pitch := ClampCameraPitch(pitchDegrees)
	if pitch > 180 {
		pitch -= 180
	}
	pitch = DegreesToRadians(pitch)
	yaw := DegreesToRadians(yawDegrees)
	horizontal := math.Cos(pitch) * distance
	target := Point3{X: targetX, Y: targetZ, Z: targetY}
	eye := Point3{
		X: target.X + math.Sin(yaw)*horizontal,
		Y: target.Y + math.Sin(pitch)*distance,
		Z: target.Z - math.Cos(yaw)*horizontal,
	}
	view := Mat4LookAt(eye, target, Point3{Y: 1})
	aspect := 1.0
	if height > 0 {
		aspect = width / height
	}
	return Mat4Multiply(Mat4Perspective(DegreesToRadians(SceneCameraFOV()), aspect, 1, 1000), view)
}

func Mat4LookAt(eye, target, up Point3) Mat4 {
	forward := Normalize3(Point3{X: target.X - eye.X, Y: target.Y - eye.Y, Z: target.Z - eye.Z})
	right := Normalize3(Cross3(up, forward))
	if right == (Point3{}) {
		right = Point3{X: 1}
	}
	cameraUp := Cross3(forward, right)
	out := Mat4Identity()
	out[0], out[1], out[2] = right.X, cameraUp.X, -forward.X
	out[4], out[5], out[6] = right.Y, cameraUp.Y, -forward.Y
	out[8], out[9], out[10] = right.Z, cameraUp.Z, -forward.Z
	out[12] = -Dot3(right, eye)
	out[13] = -Dot3(cameraUp, eye)
	out[14] = Dot3(forward, eye)
	return out
}

func Mat4Perspective(fovy, aspect, near, far float64) Mat4 {
	f := 1 / math.Tan(fovy*0.5)
	out := Mat4{}
	out[0] = f / aspect
	out[5] = f
	out[10] = (far + near) / (near - far)
	out[11] = -1
	out[14] = (2 * far * near) / (near - far)
	return out
}

func CameraPitch() float64 {
	return DefaultSceneCameraPitch
}

func CameraZoom() float64 {
	return NormalizeSceneCameraZoom(DefaultSceneCameraZoom)
}
