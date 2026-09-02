// Package compositor provides a Layer Tree for retained-mode rendering.
//
// STATUS: NOT IN PRODUCTION PIPELINE. This package is fully implemented
// and tested but not connected to desktop.draw(). The production pipeline
// uses per-boundary GPU textures (Phase 7) which bypasses the Layer Tree
// for simpler direct texture caching + blit. This package is retained as
// infrastructure for future optimizations: animated transforms on cached
// textures, opacity blending layers, clip masking without re-recording.
//
// Each [RepaintBoundary] widget creates a [PictureLayer] that owns a
// scene.Scene display list. The [Compositor] assembles all layers into
// a composed scene by REFERENCE (not copy), so when a child layer is
// re-recorded, the parent automatically sees fresh content.
//
// This is the Flutter rendering/layer.dart pattern:
//
//   - [ContainerLayer]: has children, no own content
//   - [OffsetLayer]: ContainerLayer + translation offset
//   - [PictureLayer]: owns a scene.Scene, leaf node
//   - [ClipRectLayer]: ContainerLayer + clip rectangle
//   - [OpacityLayer]: ContainerLayer + alpha blending
//
// See: ADR-007 Phase 5 (docs/dev/architecture/ADR-007-RETAINED-MODE-COMPOSITOR.md)
// Task: TASK-UI-OPT-005-compositor-integration (backlog — connect or remove)
//
// ADR-007 Phase 5 | Flutter rendering/layer.dart
package compositor
