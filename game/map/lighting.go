package maps

import (
	"hash/fnv"
	"image/color"
	"math"
	"strings"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
)

func quadNormal(verts [4]scene.Point3) scene.Point3 {
	normal := scene.Normalize3(scene.Cross3(scene.Sub3(verts[1], verts[0]), scene.Sub3(verts[2], verts[0])))
	if normal == (scene.Point3{}) {
		return scene.Point3{Y: 1}
	}
	return normal
}

type Lighting struct {
	direction scene.Point3
	Diffuse   scene.Point3
	ambient   scene.Point3
	Env       scene.Point3
}

func sceneLightingFromRSW(rsw *res.RSW) Lighting {
	longitude := 45.0
	latitude := 45.0
	diffuse := scene.Point3{X: 1, Y: 1, Z: 1}
	ambient := scene.Point3{}
	if rsw != nil {
		longitude = float64(rsw.Light.Longitude)
		latitude = float64(rsw.Light.Latitude)
		diffuse = scene.Point3{X: float64(rsw.Light.Diffuse[0]), Y: float64(rsw.Light.Diffuse[1]), Z: float64(rsw.Light.Diffuse[2])}
		ambient = scene.Point3{X: float64(rsw.Light.Ambient[0]), Y: float64(rsw.Light.Ambient[1]), Z: float64(rsw.Light.Ambient[2])}
	}
	longitude = scene.DegreesToRadians(longitude)
	latitude = scene.DegreesToRadians(latitude)
	dir := scene.Normalize3(scene.Point3{
		X: -math.Cos(longitude) * math.Sin(latitude),
		Y: -math.Cos(latitude),
		Z: -math.Sin(longitude) * math.Sin(latitude),
	})
	if dir == (scene.Point3{}) {
		dir = scene.Normalize3(scene.Point3{X: -0.5, Y: -0.7, Z: -0.5})
	}
	diffuse = clampUnitPoint(diffuse)
	ambient = clampUnitPoint(ambient)
	env := scene.Point3{
		X: 1 - (1-ambient.X)*(1-diffuse.X),
		Y: 1 - (1-ambient.Y)*(1-diffuse.Y),
		Z: 1 - (1-ambient.Z)*(1-diffuse.Z),
	}
	return Lighting{
		direction: dir,
		Diffuse:   diffuse,
		ambient:   ambient,
		Env:       clampUnitPoint(env),
	}
}

func (m *System) Lighting(rsw *res.RSW) Lighting {
	lighting := sceneLightingFromRSW(rsw)
	if !m.Night {
		return lighting
	}
	lighting.Diffuse.X = math.Min(lighting.Diffuse.X, 0.5)
	lighting.Diffuse.Y = math.Min(lighting.Diffuse.Y, 0.5)
	lighting.Env = scene.Point3{
		X: 1 - (1-lighting.ambient.X)*(1-lighting.Diffuse.X),
		Y: 1 - (1-lighting.ambient.Y)*(1-lighting.Diffuse.Y),
		Z: 1 - (1-lighting.ambient.Z)*(1-lighting.Diffuse.Z),
	}
	return lighting
}

func (l Lighting) groundScale(normal scene.Point3) scene.Point3 {
	weight := math.Max(scene.Dot3(scene.Normalize3(normal), l.direction), 0)
	return scene.Point3{
		X: scene.ClampUnit(l.ambient.X+l.Diffuse.X*weight) * scene.ClampUnit(l.Env.X),
		Y: scene.ClampUnit(l.ambient.Y+l.Diffuse.Y*weight) * scene.ClampUnit(l.Env.Y),
		Z: scene.ClampUnit(l.ambient.Z+l.Diffuse.Z*weight) * scene.ClampUnit(l.Env.Z),
	}
}

func (l Lighting) ModelScaleNormalized(normal scene.Point3) scene.Point3 {
	weight := math.Max(scene.Dot3(normal, l.direction), 0.5)
	return l.modelScaleFromWeight(weight)
}

func (l Lighting) modelScaleFromWeight(weight float64) scene.Point3 {
	return scene.Point3{
		X: scene.ClampUnit(l.ambient.X+l.Diffuse.X*weight) * scene.ClampUnit(l.Env.X),
		Y: scene.ClampUnit(l.ambient.Y+l.Diffuse.Y*weight) * scene.ClampUnit(l.Env.Y),
		Z: scene.ClampUnit(l.ambient.Z+l.Diffuse.Z*weight) * scene.ClampUnit(l.Env.Z),
	}
}

func clampUnitPoint(point scene.Point3) scene.Point3 {
	return scene.Point3{X: scene.ClampUnit(point.X), Y: scene.ClampUnit(point.Y), Z: scene.ClampUnit(point.Z)}
}

func textureColor(name string) color.RGBA {
	if name == "" {
		return color.RGBA{R: 78, G: 86, B: 78, A: 255}
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.ToLower(name)))
	value := hash.Sum32()
	return color.RGBA{
		R: 70 + uint8(value&0x3f),
		G: 80 + uint8((value>>8)&0x4f),
		B: 68 + uint8((value>>16)&0x4f),
		A: 255,
	}
}
