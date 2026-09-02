package widget

import (
	"testing"

	"github.com/gogpu/ui/geometry"
)

// BenchmarkDrawTree_FlatTree measures draw tree traversal for a flat widget tree
// (typical hello example: ~20 widgets).
func BenchmarkDrawTree_FlatTree(b *testing.B) {
	root := newDrawTrackingWidget()
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	for i := 0; i < 20; i++ {
		child := newDrawTrackingWidget()
		child.SetBounds(geometry.NewRect(0, float32(i*30), 800, 30))
		root.AddChild(child)
	}

	canvas := &noopCanvas{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.SetNeedsRedraw(true)
		DrawTree(root, nil, canvas)
	}
}

// BenchmarkDrawTree_DeepTree measures draw tree traversal for a deep widget tree
// (typical gallery: ~100+ widgets in nested boxes).
func BenchmarkDrawTree_DeepTree(b *testing.B) {
	root := newDrawTrackingWidget()
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	current := root
	for i := 0; i < 10; i++ {
		box := newDrawTrackingWidget()
		box.SetBounds(geometry.NewRect(0, 0, 800, 600))
		current.AddChild(box)
		for j := 0; j < 10; j++ {
			leaf := newDrawTrackingWidget()
			leaf.SetBounds(geometry.NewRect(0, float32(j*30), 800, 30))
			box.AddChild(leaf)
		}
		current = box
	}

	canvas := &noopCanvas{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.SetNeedsRedraw(true)
		DrawTree(root, nil, canvas)
	}
}

// BenchmarkCollectDirtyStats_LargeTree measures dirty stats collection
// for a large tree (worst case for collector).
func BenchmarkCollectDirtyStats_LargeTree(b *testing.B) {
	root := newDrawTrackingWidget()
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	for i := 0; i < 100; i++ {
		child := newDrawTrackingWidget()
		child.SetBounds(geometry.NewRect(0, float32(i*10), 800, 10))
		if i%3 == 0 {
			child.SetNeedsRedraw(true)
		}
		root.AddChild(child)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectDrawStats(root)
	}
}
