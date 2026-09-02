package layout

import (
	"testing"

	"github.com/gogpu/ui/geometry"
)

func BenchmarkFlexCompute(b *testing.B) {
	layout := &FlexLayout{}
	tree := newTestTree()
	root := NodeID(1)

	for i := 0; i < 10; i++ {
		child := NodeID(10 + i)
		tree.AddChild(root, child)
		tree.SetPreferredSize(child, geometry.Size{
			Width:  float32(40 + i),
			Height: float32(20 + i%3),
		})
	}

	tree.SetStyle(root, &Style{
		FlexDirection:  FlexRow,
		JustifyContent: JustifyStart,
		AlignItems:     AlignItemsStart,
		Gap:            4,
	})

	available := geometry.Size{Width: 800, Height: 200}
	layout.Compute(tree, root, available)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layout.Compute(tree, root, available)
	}
}

func BenchmarkGridCompute(b *testing.B) {
	layout := &GridLayout{
		Columns: []GridTrack{
			FractionTrack(1),
			FractionTrack(1),
			FractionTrack(1),
		},
		ColumnGap: 8,
		RowGap:    6,
	}
	tree := newTestTree()
	root := NodeID(1)

	for i := 0; i < 10; i++ {
		child := NodeID(10 + i)
		tree.AddChild(root, child)
		tree.SetPreferredSize(child, geometry.Size{
			Width:  60,
			Height: float32(24 + i%4),
		})
	}

	available := geometry.Size{Width: 900, Height: 600}
	layout.Compute(tree, root, available)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layout.Compute(tree, root, available)
	}
}

func BenchmarkStackCompute(b *testing.B) {
	cases := []struct {
		name      string
		layout    *StackLayout
		available geometry.Size
	}{
		{
			name:      "vertical",
			layout:    &StackLayout{Direction: StackVertical, Alignment: StackAlignStart, Spacing: 4},
			available: geometry.Size{Width: 400, Height: 800},
		},
		{
			name:      "horizontal",
			layout:    &StackLayout{Direction: StackHorizontal, Alignment: StackAlignStart, Spacing: 4},
			available: geometry.Size{Width: 800, Height: 400},
		},
		{
			name:      "zstack",
			layout:    &StackLayout{Direction: StackZ, Alignment: StackAlignCenter},
			available: geometry.Size{Width: 600, Height: 600},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			tree := newTestTree()
			root := NodeID(1)
			for i := 0; i < 10; i++ {
				child := NodeID(10 + i)
				tree.AddChild(root, child)
				tree.SetPreferredSize(child, geometry.Size{
					Width:  float32(30 + i),
					Height: float32(15 + i%5),
				})
			}

			tc.layout.Compute(tree, root, tc.available)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tc.layout.Compute(tree, root, tc.available)
			}
		})
	}
}
