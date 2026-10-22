package testutil

import "github.com/kivutar/goro/game/scene"

func Projection(width, height int, targetX, targetY, targetZ float64) scene.Projection {
	return ProjectionYaw(width, height, targetX, targetY, targetZ, scene.SceneCameraYaw())
}

func ProjectionYaw(width, height int, targetX, targetY, targetZ, yaw float64) scene.Projection {
	return ProjectionYawZoom(width, height, targetX, targetY, targetZ, yaw, scene.CameraZoom())
}

func ProjectionYawZoom(width, height int, targetX, targetY, targetZ, yaw, zoom float64) scene.Projection {
	return scene.NewProjection(width, height, targetX, targetY, targetZ, yaw, scene.CameraPitch(), zoom)
}
