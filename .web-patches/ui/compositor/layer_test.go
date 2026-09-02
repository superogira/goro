package compositor

import (
	"testing"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/geometry"
)

func TestNewOffsetLayer(t *testing.T) {
	l := NewOffsetLayer(geometry.Pt(10, 20))

	if l.Offset() != (geometry.Point{X: 10, Y: 20}) {
		t.Errorf("offset = %v, want (10, 20)", l.Offset())
	}
	if !l.NeedsCompositing() {
		t.Error("new layer should need compositing")
	}
	if l.Parent() != nil {
		t.Error("root layer should have nil parent")
	}
}

func TestOffsetLayer_AppendRemove(t *testing.T) {
	parent := NewOffsetLayer(geometry.Point{})
	child := NewPictureLayer()

	parent.Append(child)

	if len(parent.Children()) != 1 {
		t.Fatalf("children count = %d, want 1", len(parent.Children()))
	}
	if child.Parent() != parent {
		t.Error("child.Parent() should be parent after Append")
	}

	parent.Remove(child)

	if len(parent.Children()) != 0 {
		t.Fatalf("children count = %d, want 0 after Remove", len(parent.Children()))
	}
	if child.Parent() != nil {
		t.Error("child.Parent() should be nil after Remove")
	}
}

func TestOffsetLayer_RemoveAll(t *testing.T) {
	parent := NewOffsetLayer(geometry.Point{})
	c1 := NewPictureLayer()
	c2 := NewPictureLayer()

	parent.Append(c1)
	parent.Append(c2)
	parent.RemoveAll()

	if len(parent.Children()) != 0 {
		t.Errorf("children count = %d after RemoveAll", len(parent.Children()))
	}
	if c1.Parent() != nil || c2.Parent() != nil {
		t.Error("children should have nil parent after RemoveAll")
	}
}

func TestPictureLayer_DirtyLifecycle(t *testing.T) {
	l := NewPictureLayer()

	if !l.IsDirty() {
		t.Error("new PictureLayer should be dirty")
	}
	if l.Picture() != nil {
		t.Error("new PictureLayer should have nil picture")
	}

	s := scene.NewScene()
	l.SetPicture(s)
	l.ClearDirty()

	if l.IsDirty() {
		t.Error("should be clean after ClearDirty")
	}
	if l.Picture() != s {
		t.Error("Picture() should return set scene")
	}

	l.MarkDirty()

	if !l.IsDirty() {
		t.Error("should be dirty after MarkDirty")
	}
}

func TestPictureLayer_SetPictureMarksCompositing(t *testing.T) {
	l := NewPictureLayer()
	l.ClearNeedsCompositing()

	s := scene.NewScene()
	l.SetPicture(s)

	if !l.NeedsCompositing() {
		t.Error("SetPicture should mark NeedsCompositing")
	}
}

func TestClipRectLayer_Basic(t *testing.T) {
	clip := geometry.NewRect(10, 10, 100, 100)
	l := NewClipRectLayer(clip)

	if l.ClipRect() != clip {
		t.Errorf("clip = %v, want %v", l.ClipRect(), clip)
	}

	child := NewPictureLayer()
	l.Append(child)

	if len(l.Children()) != 1 {
		t.Fatalf("children = %d, want 1", len(l.Children()))
	}
}

func TestOpacityLayer_Basic(t *testing.T) {
	l := NewOpacityLayer(0.5)

	if l.Opacity() != 0.5 {
		t.Errorf("opacity = %f, want 0.5", l.Opacity())
	}

	l.SetOpacity(0.8)

	if l.Opacity() != 0.8 {
		t.Errorf("opacity = %f, want 0.8", l.Opacity())
	}
	if !l.NeedsCompositing() {
		t.Error("SetOpacity should mark NeedsCompositing")
	}
}

func TestSetOffset_MarksNeedsCompositing(t *testing.T) {
	l := NewOffsetLayer(geometry.Point{})
	l.ClearNeedsCompositing()

	l.SetOffset(geometry.Pt(50, 50))

	if !l.NeedsCompositing() {
		t.Error("SetOffset should mark NeedsCompositing")
	}
}

func TestLayerTree_ThreeLevels(t *testing.T) {
	root := NewOffsetLayer(geometry.Point{})

	buttons := NewOffsetLayer(geometry.Pt(0, 100))
	buttonsPic := NewPictureLayer()
	buttons.Append(buttonsPic)
	root.Append(buttons)

	spinner := NewOffsetLayer(geometry.Pt(200, 400))
	spinnerPic := NewPictureLayer()
	spinner.Append(spinnerPic)
	root.Append(spinner)

	if len(root.Children()) != 2 {
		t.Fatalf("root children = %d, want 2", len(root.Children()))
	}

	children := root.Children()
	if children[0] != buttons {
		t.Error("first child should be buttons layer")
	}
	if children[1] != spinner {
		t.Error("second child should be spinner layer")
	}
}

// --- Phase D: PictureLayerImpl extended fields ---

func TestPictureLayer_BoundaryCacheKey(t *testing.T) {
	l := NewPictureLayer()
	l.SetBoundaryCacheKey(42)
	if l.BoundaryCacheKey() != 42 {
		t.Errorf("BoundaryCacheKey = %d, want 42", l.BoundaryCacheKey())
	}
}

func TestPictureLayer_IsRoot(t *testing.T) {
	l := NewPictureLayer()
	if l.IsRoot() {
		t.Error("new PictureLayer should not be root")
	}
	l.SetRoot(true)
	if !l.IsRoot() {
		t.Error("PictureLayer should be root after SetRoot(true)")
	}
}

func TestPictureLayer_Size(t *testing.T) {
	l := NewPictureLayer()
	l.SetSize(800, 600)
	w, h := l.Size()
	if w != 800 || h != 600 {
		t.Errorf("Size = (%d, %d), want (800, 600)", w, h)
	}
}

func TestPictureLayer_ScreenOrigin(t *testing.T) {
	l := NewPictureLayer()
	if l.IsScreenOriginValid() {
		t.Error("new PictureLayer should have invalid ScreenOrigin")
	}
	l.SetScreenOrigin(geometry.Pt(100, 200))
	origin := l.ScreenOrigin()
	if origin.X != 100 || origin.Y != 200 {
		t.Errorf("ScreenOrigin = %v, want (100, 200)", origin)
	}
	if !l.IsScreenOriginValid() {
		t.Error("ScreenOrigin should be valid after SetScreenOrigin")
	}
}

func TestPictureLayer_PictureClipRect(t *testing.T) {
	l := NewPictureLayer()
	if l.HasPictureClip() {
		t.Error("new PictureLayer should not have clip")
	}
	clip := geometry.NewRect(10, 20, 200, 300)
	l.SetPictureClipRect(clip)
	if !l.HasPictureClip() {
		t.Error("PictureLayer should have clip after SetPictureClipRect")
	}
	if l.PictureClipRect() != clip {
		t.Errorf("PictureClipRect = %v, want %v", l.PictureClipRect(), clip)
	}
}

func TestPictureLayer_SceneVersion(t *testing.T) {
	l := NewPictureLayer()
	if l.SceneVersion() != 0 {
		t.Error("new PictureLayer should have SceneVersion 0")
	}
	l.SetSceneVersion(5)
	if l.SceneVersion() != 5 {
		t.Errorf("SceneVersion = %d, want 5", l.SceneVersion())
	}
}
