//go:build js && wasm

package ui

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

// The minimap renders as page DOM on the web build (crisp name text, zero
// canvas raster): the map thumbnail travels once per map as a PNG data URL,
// markers as map-fraction coordinates, and dragging lives entirely in the
// page. Native builds keep the canvas window.

// minimapWebEnabled reports whether the page provides the DOM minimap.
func minimapWebEnabled() bool {
	return js.Global().Get("goroMinimapSync").Type() == js.TypeFunction
}

// minimapWebMapData encodes the scaled thumbnail once per map+size and
// caches the data URL.
func (m *Minimap) minimapWebMapData(size int) string {
	img := m.scaledImage(size)
	if img == nil {
		return ""
	}
	key := fmt.Sprintf("%s:%d", m.mapName, size)
	if m.webMapKey == key && m.webMapData != "" {
		return m.webMapData
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	m.webMapKey = key
	m.webMapData = url
	return url
}

// minimapWebSync pushes the full minimap state when its visual key changes.
// Compass markers expire, so a one-second time bucket keeps them pruning.
func (m *Minimap) minimapWebSync(ctx Context, now time.Time) {
	if ctx.World == nil {
		return
	}
	width, height := ctx.ScreenSize()
	x, y, w, h := minimapBounds(width, height)
	size := minimapContentMapSize(w, h)

	player := ctx.World.Player
	mapW, mapH := 1, 1
	if ctx.World.GAT != nil && ctx.World.GAT.Width > 0 && ctx.World.GAT.Height > 0 {
		mapW, mapH = ctx.World.GAT.Width, ctx.World.GAT.Height
	}
	bucket := int64(0)
	if len(m.compass) > 0 {
		bucket = now.Unix()
	}
	key := fmt.Sprintf("open=%t|map=%s|%d,%d,%d|%d,%d,%d|rev=%d,%d,%d|t=%d",
		!m.hidden, m.mapName, x, y, size,
		player.X, player.Y, player.Dir,
		m.compassRevision, m.guildRevision, m.bossRevision, bucket)
	if key == m.webSyncKey {
		return
	}
	m.webSyncKey = key

	fn := js.Global().Get("goroMinimapSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("open", !m.hidden)
	obj.Set("x", x)
	obj.Set("y", y)
	obj.Set("size", size)
	obj.Set("name", ctx.World.MapName)
	obj.Set("mapData", m.minimapWebMapData(size))

	// Cell Y grows southward while the minimap image is drawn north-up
	// (the canvas path maps y from the bottom: projected.y + h - y*frac).
	flipY := func(cellY int) float64 {
		return 1 - float64(cellY)/float64(mapH)
	}
	playerObj := js.Global().Get("Object").New()
	playerObj.Set("fx", float64(player.X)/float64(mapW))
	playerObj.Set("fy", flipY(player.Y))
	playerObj.Set("dir", player.Dir)
	obj.Set("player", playerObj)
	obj.Set("coords", fmt.Sprintf("X:%d Y:%d", player.X, player.Y))

	compass := js.Global().Get("Array").New(len(m.compass))
	i := 0
	for _, marker := range m.compass {
		entry := js.Global().Get("Object").New()
		entry.Set("fx", float64(marker.x)/float64(mapW))
		entry.Set("fy", flipY(marker.y))
		entry.Set("css", fmt.Sprintf("#%02x%02x%02x", marker.color.R, marker.color.G, marker.color.B))
		compass.SetIndex(i, entry)
		i++
	}
	obj.Set("compass", compass)

	guild := js.Global().Get("Array").New(len(m.guild))
	i = 0
	for _, marker := range m.guild {
		entry := js.Global().Get("Object").New()
		entry.Set("fx", float64(marker.x)/float64(mapW))
		entry.Set("fy", flipY(marker.y))
		guild.SetIndex(i, entry)
		i++
	}
	obj.Set("guild", guild)

	boss := js.Undefined()
	if m.boss != nil {
		bossObj := js.Global().Get("Object").New()
		bossObj.Set("fx", float64(m.boss.x)/float64(mapW))
		bossObj.Set("fy", flipY(m.boss.y))
		boss = bossObj
	}
	obj.Set("boss", boss)
	fn.Invoke(obj)
}

// minimapWebSyncClosed hides the DOM minimap (map screen without a world).
func (m *Minimap) minimapWebSyncClosed() {
	if m.webSyncKey == "closed" {
		return
	}
	m.webSyncKey = "closed"
	fn := js.Global().Get("goroMinimapSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("open", false)
	fn.Invoke(obj)
}

// minimapWebFractionKey is used by tests to verify formatting.
func minimapWebFractionKey(v int, max int) string {
	return strconv.FormatFloat(float64(v)/float64(max), 'f', 4, 64)
}

var _ = strings.TrimSpace
