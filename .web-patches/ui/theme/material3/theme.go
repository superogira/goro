package material3

import (
	"github.com/gogpu/ui/theme"
	"github.com/gogpu/ui/widget"
)

// Theme provides Material 3 (Material You) design tokens.
//
// A Theme contains the complete set of design tokens needed to style
// a Material 3 application: colors, typography, and shape. It is
// generated from a single seed color, which is used to derive a full
// harmonious color scheme.
//
// Create a theme with [New]:
//
//	theme := material3.New(widget.Hex(0x6750A4))
//	primary := theme.Colors.Primary
//	fontSize := theme.Typography.BodyMedium.FontSize
//	radius := theme.Shape.Medium
type Theme struct {
	// Colors holds the Material 3 color scheme derived from the seed color.
	Colors ColorScheme

	// Typography holds the Material 3 type scale.
	Typography TypeScale

	// Shape holds the Material 3 corner radius scale.
	Shape ShapeScale

	// dark indicates whether this theme uses a dark color scheme.
	dark bool
}

// New creates a Material 3 theme from a seed color.
//
// The seed color drives the entire color scheme. Material 3 derives
// primary, secondary, tertiary, neutral, and error palettes from
// this single seed using HCT (Hue, Chroma, Tone) color science.
//
// By default, the theme uses a light color scheme. Use [NewDark] for
// a dark scheme, or access [Light] and [Dark] functions to generate
// color schemes independently.
func New(seedColor widget.Color) *Theme {
	return &Theme{
		Colors:     Light(seedColor),
		Typography: DefaultTypeScale(),
		Shape:      DefaultShapeScale(),
	}
}

// NewDark creates a Material 3 theme with a dark color scheme from a seed color.
//
// This is equivalent to New but uses dark mode tonal mappings.
func NewDark(seedColor widget.Color) *Theme {
	return &Theme{
		Colors:     Dark(seedColor),
		Typography: DefaultTypeScale(),
		Shape:      DefaultShapeScale(),
		dark:       true,
	}
}

// IsDark returns true if this theme uses a dark color scheme.
func (t *Theme) IsDark() bool {
	return t.dark
}

// AsTheme converts the Material 3 theme to a [theme.Theme] for use with
// the generic theme system. This maps M3 color roles to the shared
// [theme.ColorPalette] structure, preserving background, surface, and
// on-color relationships.
//
// Use this when you need to pass an M3 theme to APIs that accept
// [*theme.Theme], such as [app.WithTheme] or [app.App.SetTheme]:
//
//	m3 := material3.New(widget.Hex(0x6750A4))
//	uiApp := app.New(app.WithTheme(m3.AsTheme()))
func (t *Theme) AsTheme() *theme.Theme {
	cs := t.Colors
	mode := theme.ModeLight
	name := "Material 3"
	shadows := theme.DefaultShadowsLight()
	if t.dark {
		mode = theme.ModeDark
		name = "Material 3 Dark"
		shadows = theme.DefaultShadowsDark()
	}

	return &theme.Theme{
		Name: name,
		Mode: mode,
		Colors: theme.ColorPalette{
			Primary:        cs.Primary,
			PrimaryLight:   cs.PrimaryContainer,
			PrimaryDark:    cs.OnPrimaryContainer,
			Secondary:      cs.Secondary,
			SecondaryLight: cs.SecondaryContainer,
			SecondaryDark:  cs.OnSecondaryContainer,
			Background:     cs.Background,
			Surface:        cs.Surface,
			SurfaceVariant: cs.SurfaceVariant,
			Error:          cs.Error,
			Warning:        cs.Error, // M3 has no dedicated warning role
			Success:        cs.Tertiary,
			Info:           cs.Primary,
			OnPrimary:      cs.OnPrimary,
			OnSecondary:    cs.OnSecondary,
			OnBackground:   cs.OnBackground,
			OnSurface:      cs.OnSurface,
			OnError:        cs.OnError,
			Divider:        cs.OutlineVariant,
			Outline:        cs.Outline,
			Shadow:         widget.RGBA(0, 0, 0, 0.20),
		},
		Typography: theme.DefaultTypography(),
		Spacing:    theme.DefaultSpacing(),
		Shadows:    shadows,
		Radii:      theme.DefaultRadii(),
		Extensions: make(map[string]any),
	}
}

// OnSurface returns the default text/icon color for surface backgrounds.
//
// In Material 3, this is the OnSurface color role derived from the
// neutral tonal palette. It provides the highest contrast on surface
// backgrounds for body text and icons.
func (t *Theme) OnSurface() widget.Color {
	return t.Colors.OnSurface
}
