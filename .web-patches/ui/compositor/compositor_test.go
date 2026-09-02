package compositor

import (
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/geometry"
)

var (
	red  = scene.SolidBrush(gg.RGBA{R: 1, A: 1})
	blue = scene.SolidBrush(gg.RGBA{B: 1, A: 1})
	gray = scene.SolidBrush(gg.RGBA{R: 0.5, G: 0.5, B: 0.5, A: 1})
)

func rectScene(brush scene.Brush, w, h float32) *scene.Scene {
	s := scene.NewScene()
	s.Fill(scene.FillNonZero, scene.IdentityAffine(), brush,
		scene.NewRectShape(0, 0, w, h))
	return s
}

// --- Bug prevention: composition by reference ---

// TestCompose_ChildReRecord_ParentSeesUpdate catches the root cause of
// the spinner freeze bug: Scene.Append COPIES data. If Compose cached
// or reused the old composed scene without re-walking, child updates
// would be invisible. This test fails if Compose returns stale content.
func TestCompose_ChildReRecord_ParentSeesUpdate(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	staticPic := NewPictureLayer()
	staticPic.SetPicture(rectScene(gray, 800, 40))
	root.Append(staticPic)

	spinnerPic := NewPictureLayer()
	spinnerPic.SetPicture(rectScene(red, 48, 48))
	spinnerPic.SetOffset(geometry.Pt(100, 200))
	root.Append(spinnerPic)

	v1 := c.Compose(root).Version()

	// Spinner re-records (next animation frame).
	spinnerPic.SetPicture(rectScene(blue, 48, 48))

	v2 := c.Compose(root).Version()

	if v2 <= v1 {
		t.Errorf("Compose after child re-record: version v2=%d <= v1=%d; "+
			"composed scene must be rebuilt, not cached", v2, v1)
	}
}

// TestCompose_10ConsecutiveFrames simulates 10 animation frames where
// spinner re-records each time. Every frame must produce a NEW composed
// scene. If any two consecutive frames have equal versions, animation
// is frozen (the bug we're fixing).
func TestCompose_10ConsecutiveFrames(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	staticPic := NewPictureLayer()
	staticPic.SetPicture(rectScene(gray, 800, 600))
	root.Append(staticPic)

	spinnerPic := NewPictureLayer()
	spinnerPic.SetOffset(geometry.Pt(400, 300))
	root.Append(spinnerPic)

	var prevVersion uint64
	for frame := 0; frame < 10; frame++ {
		spinnerPic.SetPicture(rectScene(red, 48, 48))
		composed := c.Compose(root)
		v := composed.Version()

		if frame > 0 && v <= prevVersion {
			t.Fatalf("frame %d: version %d <= previous %d; animation frozen", frame, v, prevVersion)
		}
		if composed.IsEmpty() {
			t.Fatalf("frame %d: composed scene is empty", frame)
		}
		prevVersion = v
	}
}

// TestCompose_StaticLayerNotReRecorded verifies that static content
// is NOT re-recorded during compose. Only spinner re-records.
// If static picture pointer changes, we're doing unnecessary work.
func TestCompose_StaticLayerNotReRecorded(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	staticScene := rectScene(gray, 800, 600)
	staticPic := NewPictureLayer()
	staticPic.SetPicture(staticScene)
	root.Append(staticPic)

	spinnerPic := NewPictureLayer()
	spinnerPic.SetPicture(rectScene(red, 48, 48))
	root.Append(spinnerPic)

	c.Compose(root)

	// After compose, static picture must still be the same object.
	if staticPic.Picture() != staticScene {
		t.Error("Compose must not replace static PictureLayer's scene; " +
			"only dirty layers should be re-recorded by PaintDirtyBoundaries, " +
			"not by the compositor")
	}
}

// --- Bug prevention: re-parenting ---

// TestReparent_ChildMovedBetweenParents prevents a bug where moving a
// child from parent A to parent B leaves a dangling reference in A.
func TestReparent_ChildMovedBetweenParents(t *testing.T) {
	parentA := NewOffsetLayer(geometry.Point{})
	parentB := NewOffsetLayer(geometry.Point{})
	child := NewPictureLayer()

	parentA.Append(child)
	if child.Parent() != parentA {
		t.Fatal("child should belong to parentA")
	}

	parentA.Remove(child)
	parentB.Append(child)

	if child.Parent() != parentB {
		t.Error("child.Parent() should be parentB after re-parenting")
	}
	if len(parentA.Children()) != 0 {
		t.Error("parentA should have 0 children after child was removed")
	}
	if len(parentB.Children()) != 1 {
		t.Error("parentB should have 1 child")
	}

	// Compose should only include child in parentB, not parentA.
	c := New()
	child.SetPicture(rectScene(red, 10, 10))
	child.SetOffset(geometry.Pt(50, 50))

	rootA := NewOffsetLayer(geometry.Point{})
	rootA.Append(parentA)
	resultA := c.Compose(rootA)
	if !resultA.IsEmpty() {
		t.Error("parentA tree should produce empty scene (child was removed)")
	}

	rootB := NewOffsetLayer(geometry.Point{})
	rootB.Append(parentB)
	resultB := c.Compose(rootB)
	if resultB.IsEmpty() {
		t.Error("parentB tree should produce non-empty scene (child present)")
	}
}

// --- Bug prevention: nil/empty handling ---

// TestCompose_NilPicture prevents crash when PictureLayer has no scene.
func TestCompose_NilPicture(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	pic := NewPictureLayer()
	// Intentionally do NOT set a picture.
	root.Append(pic)

	result := c.Compose(root)
	if result == nil {
		t.Fatal("Compose should not return nil even with nil pictures")
	}
}

// TestCompose_EmptyScene prevents composed scene from containing
// garbage when picture's scene was Reset but not re-filled.
func TestCompose_EmptyScene(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	pic := NewPictureLayer()
	s := scene.NewScene()
	s.Reset() // empty scene
	pic.SetPicture(s)
	root.Append(pic)

	result := c.Compose(root)
	if !result.IsEmpty() {
		t.Error("composed scene should be empty when all pictures are empty")
	}
}

// TestCompose_NilRoot prevents crash on nil root.
func TestCompose_NilRoot(t *testing.T) {
	c := New()
	result := c.Compose(nil)
	if result == nil {
		t.Fatal("Compose(nil) must return empty scene, not nil")
	}
}

// --- Bug prevention: offset accumulation ---

// TestCompose_OffsetAccumulation verifies exact pixel positions through
// 3 levels of nesting. A bug in offset accumulation shifts ALL content.
func TestCompose_OffsetAccumulation(t *testing.T) {
	c := New()

	root := NewOffsetLayer(geometry.Pt(10, 20))
	mid := NewOffsetLayer(geometry.Pt(30, 40))
	root.Append(mid)

	pic := NewPictureLayer()
	pic.SetPicture(rectScene(red, 50, 50))
	pic.SetOffset(geometry.Pt(5, 5))
	mid.Append(pic)

	result := c.Compose(root)
	bounds := result.Bounds()

	// Expected: (10+30+5, 20+40+5) = (45, 65)
	wantX, wantY := float32(45), float32(65)

	if bounds.MinX < wantX-1 || bounds.MinX > wantX+1 {
		t.Errorf("bounds.MinX = %f, want ~%f (10+30+5)", bounds.MinX, wantX)
	}
	if bounds.MinY < wantY-1 || bounds.MinY > wantY+1 {
		t.Errorf("bounds.MinY = %f, want ~%f (20+40+5)", bounds.MinY, wantY)
	}
}

// TestCompose_ZeroOffset verifies that zero-offset layers don't shift content.
func TestCompose_ZeroOffset(t *testing.T) {
	c := New()

	root := NewOffsetLayer(geometry.Point{})
	pic := NewPictureLayer()
	pic.SetPicture(rectScene(red, 100, 100))
	root.Append(pic)

	result := c.Compose(root)
	bounds := result.Bounds()

	if bounds.MinX > 1 || bounds.MinY > 1 {
		t.Errorf("zero-offset should not shift content: bounds start at (%f, %f)",
			bounds.MinX, bounds.MinY)
	}
}

// --- Bug prevention: dirty tracking ---

// TestCompose_ClearsNeedsCompositing ensures flags are cleared after
// compose. Without this, compositor runs expensive composition every
// frame even when nothing changed.
func TestCompose_ClearsNeedsCompositing(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	pic := NewPictureLayer()
	pic.SetPicture(rectScene(red, 10, 10))
	root.Append(pic)

	c.Compose(root)

	if root.NeedsCompositing() {
		t.Error("root NeedsCompositing should be false after Compose")
	}
	if pic.NeedsCompositing() {
		t.Error("pic NeedsCompositing should be false after Compose")
	}
}

// TestCompose_ChildDirtyMarksParent verifies that dirtying a child
// marks all ancestors as needing compositing.
func TestCompose_ChildDirtyMarksParent(t *testing.T) {
	root := NewOffsetLayer(geometry.Point{})
	root.ClearNeedsCompositing()

	mid := NewOffsetLayer(geometry.Point{})
	root.Append(mid)
	// Append sets NeedsCompositing on root. Clear to test MarkDirty path.
	root.ClearNeedsCompositing()
	mid.ClearNeedsCompositing()

	pic := NewPictureLayer()
	mid.Append(pic)

	// pic.Append marked mid and root as needing compositing.
	if !mid.NeedsCompositing() {
		t.Error("mid should need compositing after child added")
	}
}

// --- Bug prevention: RemoveAll doesn't leak ---

// TestRemoveAll_NoDanglingParent prevents memory leak where removed
// children still reference the old parent.
func TestRemoveAll_NoDanglingParent(t *testing.T) {
	parent := NewOffsetLayer(geometry.Point{})
	children := make([]*PictureLayerImpl, 5)
	for i := range children {
		children[i] = NewPictureLayer()
		parent.Append(children[i])
	}

	parent.RemoveAll()

	for i, ch := range children {
		if ch.Parent() != nil {
			t.Errorf("child[%d].Parent() != nil after RemoveAll (dangling reference)", i)
		}
	}
}

// --- Bug prevention: deep nesting ---

// TestCompose_DeepNesting100 prevents stack overflow on deep layer trees.
func TestCompose_DeepNesting100(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	current := root
	for i := 0; i < 100; i++ {
		child := NewOffsetLayer(geometry.Pt(1, 1))
		current.Append(child)
		current = child
	}

	pic := NewPictureLayer()
	pic.SetPicture(rectScene(red, 10, 10))
	current.Append(pic)

	result := c.Compose(root)

	if result.IsEmpty() {
		t.Error("100-deep layer tree should produce non-empty scene")
	}

	bounds := result.Bounds()
	// 100 levels × (1,1) offset = (100, 100)
	if bounds.MinX < 99 || bounds.MinX > 101 {
		t.Errorf("100-deep offset: bounds.MinX = %f, want ~100", bounds.MinX)
	}
}

// --- Functional: basic operations ---

func TestCompose_EmptyTree(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})
	result := c.Compose(root)
	if !result.IsEmpty() {
		t.Error("empty tree should produce empty scene")
	}
}

func TestCompose_SinglePicture(t *testing.T) {
	c := New()
	root := NewOffsetLayer(geometry.Point{})

	pic := NewPictureLayer()
	pic.SetPicture(rectScene(red, 100, 50))
	root.Append(pic)

	result := c.Compose(root)
	if result.IsEmpty() {
		t.Error("single picture should produce non-empty scene")
	}
}
