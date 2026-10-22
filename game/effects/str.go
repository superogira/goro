package effects

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const StrEffectDepthBias = 0.02

func (m *System) drawSTREffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, worldX, worldY, worldZ float64, now time.Time) bool {
	str := m.LoadWorldEffectSTR(ctx.Resources, ResolveEffectSTRFile(component, effect, lessEffectsEnabled(ctx)), component.TexturePath)
	if str == nil {
		return false
	}
	fps := str.FPS
	if fps <= 0 {
		fps = 60
	}
	keyIndex := StrEffectKeyIndex(effect.Starts, now, fps, str.MaxKey, component.Repeat)
	drawn := false
	for _, layer := range str.Layers {
		anim, ok := calculateSTRAnimation(layer, keyIndex)
		if !ok {
			continue
		}
		if math.IsNaN(float64(anim.AniFrame)) || math.IsInf(float64(anim.AniFrame), 0) {
			continue
		}
		textureIndex := int(anim.AniFrame)
		if textureIndex < 0 || textureIndex >= len(layer.Textures) {
			continue
		}
		texture := m.strEffectTexture(ctx.Resources, layer.Textures[textureIndex])
		if texture == nil {
			continue
		}
		anim.Pos[0] += float32(component.spriteXOffset)
		anim.Pos[1] += float32(component.SpriteYOffset)
		drawSTRAnimation(screen, projection, texture, worldX, worldY, worldZ, anim, component.AttachedEntity)
		drawn = true
	}
	return drawn
}

func StrEffectKeyIndex(starts, now time.Time, fps int, maxKey int, repeat bool) float64 {
	if fps <= 0 {
		fps = 60
	}
	keyIndex := float64(now.Sub(starts)) / float64(time.Second) * float64(fps)
	if repeat && maxKey > 0 {
		keyIndex = math.Mod(keyIndex, float64(maxKey))
		if keyIndex < 0 {
			keyIndex += float64(maxKey)
		}
	}
	return keyIndex
}

func ResolveEffectSTRFile(component Component, effect Instance, lessEffects bool) string {
	if lessEffects && component.StrMinFile != "" {
		return component.StrMinFile
	}
	if component.StrFile == "" || !strings.Contains(component.StrFile, "%d") || component.StrRandMax < component.StrRandMin || component.StrRandMin <= 0 {
		return component.StrFile
	}
	span := component.StrRandMax - component.StrRandMin + 1
	index := component.StrRandMin
	if span > 1 {
		value := uint32(effect.EffectID*1103515245) ^ effect.ActorID ^ uint32(effect.Starts.UnixNano())
		value ^= value >> 16
		index += int(value % uint32(span))
	}
	return fmt.Sprintf(component.StrFile, index)
}

func (m *System) LoadWorldEffectSTR(manager *res.Manager, strFile, texturePath string) *res.STR {
	if manager == nil || strFile == "" {
		return nil
	}
	if m.StrEffects == nil {
		m.StrEffects = make(map[string]*res.STR)
	}
	if m.StrEffectMiss == nil {
		m.StrEffectMiss = make(map[string]struct{})
	}
	normalized := strings.ReplaceAll(strFile, "/", "\\")
	paths := []string{"data\\texture\\effect\\" + normalized + ".str"}
	if strings.ContainsAny(strFile, `/\`) {
		paths = append([]string{"data\\texture\\" + normalized + ".str"}, paths...)
	}
	attempted := false
	for _, path := range paths {
		key := "__str_" + path + "|" + texturePath
		if str, ok := m.StrEffects[key]; ok {
			return str
		}
		if _, ok := m.StrEffectMiss[key]; ok {
			continue
		}
		attempted = true
		data, err := manager.ReadFileExact(path)
		if err != nil {
			m.StrEffectMiss[key] = struct{}{}
			continue
		}
		str, err := res.ParseSTR(data, texturePath)
		if err != nil {
			m.StrEffectMiss[key] = struct{}{}
			glog.Warnf("str effect parse failed path=%s: %v", path, err)
			return nil
		}
		m.StrEffects[key] = str
		return str
	}
	if attempted {
		glog.Warnf("str effect missing file=%s", strFile)
	}
	return nil
}

func (m *System) strEffectTexture(manager *res.Manager, path string) *render.Image {
	path = strings.TrimSpace(path)
	if manager == nil || path == "" {
		return nil
	}
	key := "__strtex_" + path
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	candidates := []string{path, strings.ReplaceAll(path, "\\", "/")}
	img, _, err := res.LoadImageExact(manager, candidates)
	if err != nil {
		m.textureMiss[key] = struct{}{}
		glog.Warnf("str effect texture missing path=%s: %v", path, err)
		return nil
	}
	texture := render.NewImageFromImage(res.ApplyEffectTransparency(img))
	m.textures[key] = texture
	return texture
}

func strEffectDuration(str *res.STR, fallback time.Duration) time.Duration {
	if str == nil || str.FPS <= 0 || str.MaxKey <= 0 {
		return fallback
	}
	duration := time.Duration(float64(str.MaxKey) / float64(str.FPS) * float64(time.Second))
	if duration <= 0 {
		return fallback
	}
	return duration + 100*time.Millisecond
}

func calculateSTRAnimation(layer res.STRLayer, keyIndex float64) (res.STRAnimation, bool) {
	animations := layer.Animations
	lastFrame := 0
	lastSource := 0
	fromID := -1
	toID := -1
	for i, anim := range animations {
		if float64(anim.Frame) <= keyIndex {
			if anim.Type == 0 {
				fromID = i
			}
			if anim.Type == 1 {
				toID = i
			}
		}
		if anim.Frame > lastFrame {
			lastFrame = anim.Frame
		}
		if anim.Type == 0 && anim.Frame > lastSource {
			lastSource = anim.Frame
		}
	}
	if fromID < 0 || (toID < 0 && float64(lastFrame) < keyIndex) {
		return res.STRAnimation{}, false
	}
	from := animations[fromID]
	var to res.STRAnimation
	hasTo := toID >= 0 && toID < len(animations)
	if hasTo {
		to = animations[toID]
	}
	delta := float32(keyIndex - float64(from.Frame))
	out := res.STRAnimation{
		SrcAlpha:  from.SrcAlpha,
		DestAlpha: from.DestAlpha,
	}
	if !hasTo || toID != fromID+1 || to.Frame != from.Frame {
		if hasTo && lastSource <= from.Frame {
			return res.STRAnimation{}, false
		}
		return from, true
	}
	out.Angle = from.Angle + to.Angle*delta
	out.AniFrame = strAnimFrame(from, to, delta, len(layer.Textures))
	for i := range out.Color {
		out.Color[i] = from.Color[i] + to.Color[i]*delta
	}
	for i := range out.Pos {
		out.Pos[i] = from.Pos[i] + to.Pos[i]*delta
	}
	for i := range out.UV {
		out.UV[i] = from.UV[i] + to.UV[i]*delta
	}
	for i := range out.XY {
		out.XY[i] = from.XY[i] + to.XY[i]*delta
	}
	return out, true
}

func strAnimFrame(from, to res.STRAnimation, delta float32, texCount int) float32 {
	switch to.AniType {
	case 1:
		return from.AniFrame + to.AniFrame*delta
	case 2:
		return minFloat32(from.AniFrame+to.Delay*delta, float32(texCount-1))
	case 3:
		count := float32(scene.MaxInt(texCount, 1))
		return float32(math.Mod(float64(from.AniFrame+to.Delay*delta), float64(count)))
	case 4:
		count := float32(scene.MaxInt(texCount, 1))
		value := float32(math.Mod(float64(from.AniFrame-to.Delay*delta), float64(count)))
		if value < 0 {
			value += count
		}
		return value
	default:
		return from.AniFrame
	}
}

func drawSTRAnimation(screen *render.Frame, projection scene.Projection, texture *render.Image, worldX, worldY, worldZ float64, anim res.STRAnimation, attached bool) {
	right, up, _, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	center := scene.Point3{X: worldX, Y: worldZ, Z: worldY}
	vertexPoint := func(x, y float32) scene.Point3 {
		dx, dy := StrAnimationLocalOffset(anim, x, y, attached)
		return scene.Add3(scene.Add3(center, scene.Mul3(right, dx)), scene.Mul3(up, dy))
	}
	tint := strAnimationTint(anim)
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	vertices := []render.Vertex3D{
		StrAnimationVertex3D(vertexPoint(anim.XY[0], anim.XY[4]), scene.UV{U: 0, V: 0}, tint, w, h, center),
		StrAnimationVertex3D(vertexPoint(anim.XY[1], anim.XY[5]), scene.UV{U: 1, V: 0}, tint, w, h, center),
		StrAnimationVertex3D(vertexPoint(anim.XY[3], anim.XY[7]), scene.UV{U: 0, V: 1}, tint, w, h, center),
		StrAnimationVertex3D(vertexPoint(anim.XY[2], anim.XY[6]), scene.UV{U: 1, V: 1}, tint, w, h, center),
	}
	screen.DrawTriangles3DOwned(vertices, scene.QuadIndices012213, texture, StrAnimationDrawOptions(anim))
}

func StrAnimationLocalOffset(anim res.STRAnimation, x, y float32, attached bool) (float64, float64) {
	const pixelRatio = 1.0 / 35.0
	angle := -float64(anim.Angle) * math.Pi / 180
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	localX := float64(x) * pixelRatio
	localY := -float64(y) * pixelRatio
	rotX := localX*cosA - localY*sinA
	rotY := localX*sinA + localY*cosA
	offsetX, offsetY := StrAnimationOffset(anim, attached)
	return rotX + offsetX, rotY + offsetY
}

func StrAnimationVertex3D(point scene.Point3, uv scene.UV, tint color.RGBA, textureWidth, textureHeight float32, depthPoint scene.Point3) render.Vertex3D {
	vertex := scene.TexturedSurfaceVertex3D(point, uv, tint, textureWidth, textureHeight)
	vertex.DepthX = float32(depthPoint.X)
	vertex.DepthY = float32(depthPoint.Y)
	vertex.DepthZ = float32(depthPoint.Z)
	return vertex
}

func StrAnimationDrawOptions(anim res.STRAnimation) *render.DrawTrianglesOptions {
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	options.Blend = StrAnimationBlend(anim)
	// robr's STR shader keeps the quad on the camera plane and subtracts
	// clip-space Z, so tall effects do not sink behind the floor at the bottom.
	options.DepthBias = StrEffectDepthBias
	// robr's StrEffect shader has uFogUse, but StrEffect.beforeRender never sets
	// it; WebGL initializes uniforms to false, so STR effects render unfogged.
	options.DisableFog = true
	return options
}

func StrAnimationBlend(anim res.STRAnimation) render.Blend {
	// reference client applies the Direct3D blend factors stored in each STR layer.
	if anim.SrcAlpha == 5 && anim.DestAlpha == 7 {
		return render.BlendSrcAlphaDstAlpha
	}
	if anim.SrcAlpha == 5 && anim.DestAlpha == 2 {
		return render.BlendLighter
	}
	if anim.DestAlpha == 2 {
		return render.BlendLighter
	}
	return render.BlendSourceOver
}

func StrAnimationOffset(anim res.STRAnimation, attached bool) (float64, float64) {
	const pixelRatio = 1.0 / 35.0
	verticalBase := -0.5
	if attached {
		verticalBase = 0
	}
	return float64(anim.Pos[0]-320) * pixelRatio, -float64(anim.Pos[1]-320)*pixelRatio + verticalBase
}

func strAnimationTint(anim res.STRAnimation) color.RGBA {
	return color.RGBA{
		R: uint8(scene.ClampFloat(float64(anim.Color[0]), 0, 1) * 255),
		G: uint8(scene.ClampFloat(float64(anim.Color[1]), 0, 1) * 255),
		B: uint8(scene.ClampFloat(float64(anim.Color[2]), 0, 1) * 255),
		A: uint8(scene.ClampFloat(float64(anim.Color[3]), 0, 1) * 255),
	}
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
