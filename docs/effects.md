# Effect code organization

Effect ownership determines the file, independently of how an effect is drawn.
A STR animation, sound-only effect, or body-color change belongs with its job just
as a procedural effect does.

- `db/effects_<job>.go` holds that job's effect definitions, including textures,
  sounds, timings, and configured calls to shared builders.
- `game/effects/<job>.go` holds custom job animation and rendering code when the
  job needs it. Jobs using the shared renderers do not need an empty runtime file.
- Shared primitives live in `game/effects/hit.go`, `ground.go`, `particles.go`,
  and `casting.go`. Related classes share files where their skills are the same,
  such as `clown_gypsy.go`.
- Non-skill effects have explicit homes such as `game/effects/environment.go`,
  `items.go`, and `aura_level99.go`; their definitions remain in `db`.

`db/effects.go` contains effect IDs, common types, and the flat `EffectSpecs`
registry. Each entry calls a named, zero-argument definition. Literal settings
and calls that configure a parameterized builder belong in the owning file.
Native and local IDs for the same effect may call the same definition.
Keep ID constants and registry entries in numeric order, with aliases beside
their resolved ID and native IDs separate from Goro-local IDs. Never renumber
an effect to fill a gap.

An inherited or reused job effect keeps its original owner: for example, Heal
and its larger variants belong to Acolyte even when Arch Bishop uses them.
A renderer supporting variants from several jobs belongs in a shared file;
its job-specific definitions stay with their jobs. Native `FuncName` strings
remain stable even when the Go renderer gets a more general name.
`game/effects/func.go` maps those names to adapters and dispatches to renderers;
the rendering implementations live with their jobs or shared primitives.

Skill-to-effect and ground-unit-to-effect mappings remain central routing tables
in `db/skills.go` and `game/skill_visuals.go`. They select effect IDs; they do not
define their visuals. Passive skills do not need placeholder effect definitions.

`game/effects` is a package in the existing Go module. Its `System` owns effect
instances, animation state, and rendering. `game/visual_bridge.go` connects it to
sound scheduling, camera shake, and actor presentation; packet handling and
authoritative actor animation remain in `game`. The texture cache stays shared
with the world so map-entry invalidation preserves its existing behavior.

Both packages use `game/scene` for projection, geometry, sprite composition, and
actor positioning. Neither `scene` nor `effects` imports `game`.

Rendering tests live beside the implementations in `game/effects`. Checks that
also exercise packets, statuses, or combat stay in the existing
`game/world_mode_*_test.go` integration suites. Shared test fixtures live in
`game/internal/testutil`; they are not used by production code.

When moving definitions, check both their ownership (including status-driven and
ground-unit effects) and their evaluated values. Preserve IDs, component order,
nil versus empty slices, timing, sounds, and draw behavior.
