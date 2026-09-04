# goro

`goro` is an open Ragnarok Online client recreation implemented in Go.

The runtime uses GoGPU/wgpu for the window and presentation path, with a modern
GPU pipeline and Vulkan support. Built 100% in Go without CGO, it is fully statically
compiled and can be easily deployed.

This project wouldn't be possible without the existence of other open source clients
like ROBrowser Legacy and Open Midgard and their reverse engineering efforts.

![goro screenshot](https://github.com/kivutar/goro/releases/download/v0.0.1/goro-20260716-164507.png)

Visit the [project website](https://kivutar.github.io/goro/) or see Goro in
action on this [YouTube playlist](https://www.youtube.com/watch?v=5qldvYi9v-U&list=PLQhSdCGUOBwc).

## Project Goals

- Faithfully reimplement the original Ragnarok Online client.
- Focus on the pre-renewal 2008 experience first.
- Stay pure Go, without CGO, so cross-compilation and deployment stay simple on
  many platforms.
- Aim for simple, readable, hackable codebase.
- Use a modern GPU pipeline through GoGPU, including Vulkan and Wayland support.
- Deliver good performance, including support for high-refresh-rate displays.
- Provide a modernized, neat themeable UI built with `gogpu/ui`.
- Keep the engine reusable for creating new MMORPGs.
- Become a drop-in replacement for `Ragexe` and `Sakexe`.

### Stretch goals:

- Provide GRF tooling.
- Provide map, sprite, and model viewers.
- Support more Ragnarok Online client versions.
- Support optional anti-cheat and security features.

## Build and Run

```sh
CGO_ENABLED=0 go build -tags nofakecgo .
./goro
```

Configuration is loaded from `goro.ini` in the current directory when the file
exists. Pass another file with `--config`:

```ini
data_dir = /home/kivutar/Téléchargements/OldRO

[window]
width = 1280
height = 720
fullscreen = false

[packet]
client_date = 20080910
profile = 23

[audio]
bgm = true
bgm_volume = 0.55

[render]
graphics_api = vulkan
vsync = true

[network]
trace = false
```

Command-line options override the ini file:

```sh
CGO_ENABLED=0 go run -tags nofakecgo . --data-dir ~/kRO --fullscreen
CGO_ENABLED=0 go run -tags nofakecgo . --config ./goro.ini --bgm=false --graphics-api vulkan
```

Useful options:

```sh
--net-trace
--packet-client-date 20211103 # only when rAthena is rebuilt for that packetver
--fullscreen
--bgm=false
--bgm-volume 0.35
--no-audio # disable BGM and SFX output entirely (useful for profiling)
--graphics-api gles # fallback if Vulkan is unavailable
--vsync=false # unlock fps
--username <username> # prefill the username in login window
--password <password> # same for password
--autologin=true # perform server connection and login on startup
--force-user-ai=true # start homunculus and mercenary in USER_AI custom mode
--script <path> # run an optional Lua character-control script in game
```

## Debugging and Profiling

### Web build (wasm) — URL parameters

Append these to the page URL, e.g. `webro/?stats=1&fps=1`. They can be combined.

| Parameter | Effect |
| --- | --- |
| `?u=<name>&p=<pass>` | Prefill login credentials |
| `?auto=1` | Connect and log in automatically on startup |
| `?char-slot=N` | Pick character slot N automatically after login (0-based); with `auto=1` the session runs from the title screen into the map with no taps |
| `?mute=1` | Disable BGM and SFX |
| `?stats=1` | Enable render stats: one-per-second frame-split lines and slow-frame warnings (>16ms) in the browser console, plus the FPS/MEM widget HUD in the bottom-right corner of the map |
| `?fps=1` | Show the engine FPS counter (top-left). Drawn straight to the frame, so it stays visible with `noui=1` |
| `?noui=1` | Skip the widget UI layer entirely once inside the map — pure 3D world with no windows. Login and character select stay playable; tap/keyboard input still works. Used to measure how much frame budget the UI raster costs |
| `?nobg=1` | Title screen without the background art |
| `?nowin=1` | Title screen without the login window |
| `?nocursor=1` | Hide the in-game cursor sprite (use the OS cursor) |

Any unique unused parameter (e.g. `?nocache=123456`) also works as a cache
buster when a heuristic cache serves a stale `index.html`.

Reading the `?stats=1` console output: each slow frame logs a breakdown such
as `split_game_draw_ms` (3D world), `split_screen_ui_ms` / `ui_canvas_ms`
(widget raster on the CPU), and `split_gpu_ms` (GPU submission). High
`ui_canvas_ms` with low `split_gpu_ms` means the frame budget is spent
rastering UI, which `noui=1` isolates.

### Native build — command-line flags

The equivalents of the web switches (also settable in `goro.ini`):

```sh
--render-stats # same as ?stats=1: frame-split logging + FPS/MEM HUD
--fps # same as ?fps=1: engine FPS counter
--no-ui # disable UI rendering entirely from boot (benchmarking only; login needs UI)
--world-debug-stats # world renderer stats
--net-trace # log raw network packets
--no-audio # disable BGM and SFX output (useful for profiling)
```

Note the difference between `--no-ui` and the web's `?noui=1`: the CLI flag
removes the UI everywhere including the login screen, while the web parameter
only suppresses it inside the map so the session stays playable.

## Getting Started

These are tutorials on how to setup a development environment.

- [Server setup](docs/rathena-setup.md)
- [Client setup](docs/client-setup.md)
- [Homunculus and mercenary support](docs/companions-20080910.md)
- [Lua bot scripting](docs/bot-scripting.md)

Runtime data is discovered from, in order:

- `--data-dir`
- `data_dir` in `goro.ini`
- current working directory

The resource manager currently looks for loose files such as:

- `data/clientinfo.xml`
- `data/sclientinfo.xml`
- `clientinfo.xml`
- `sclientinfo.xml`
- `System/clientinfo.xml`
- `System/sclientinfo.xml`

## Current Scope

Currently implemented (not a claim of complete reference-client parity):

 * Login
 * Character selection
 * Character creation
 * Character deletion
 * Maps display
   * Water
   * Map sounds
   * Lightmaps
   * RSW fog with reference-client near/far behavior
   * Animated models
   * Weather effects
   * Map-specific effects such as Yuno clouds, pillars, and fireworks
   * Indoors
   * Granny 3D NPC models
   * Black-covered map loading with first-frame prewarming and fade-in
 * Camera
   * Smooth character following and zoom
   * Rotation and bounded outdoor tilt
   * Indoor and map-authored viewpoint locks
   * Outdoor zoom and rotation restoration after locked maps
 * Battle and Gameplay
   * Enemies
   * Classic PvP map targeting and rank counter
   * [War of Emperium (2008 FE/SE client behavior)](docs/woe-20080910.md)
   * Path finding
   * Continuous held-click walking
   * Drops
   * Playable characters animation chain
   * Attack-ready stance
   * Jobs
     * Novice
     * Super Novice
     * First jobs
       * Swordman
       * Magician
       * Archer
       * Acolyte
       * Thief
       * Merchant
     * Second jobs
       * Knight and Crusader
       * Wizard and Sage
       * Hunter, Bard, and Dancer
       * Priest and Monk
       * Blacksmith and Alchemist
       * Assassin and Rogue
     * Transcendent jobs
       * Lord Knight and Paladin
       * High Wizard and Professor
       * Sniper, Clown, and Gypsy
       * High Priest and Champion
       * Whitesmith and Creator
       * Assassin Cross and Stalker
     * Expanded jobs, including baby variants
       * Taekwon, Star Gladiator, and Soul Linker
       * Gunslinger and Ninja
   * Skill effects
     * Ground skill units and cast markers
     * Song and dance effects
   * Skill casting
   * Walk cancellation
   * Casting cancellation
   * Cursor snap
   * Noshift
   * Noctrl
   * Item drops
   * Item identification
   * Card composition
   * Trading
   * Vending
   * Show equipment
   * Alchemist crafting
   * Blacksmith repair and weapon refinement
   * Guilds
     * Creation and invitations
     * Member and position management
     * Leaving, member expulsion, and guild disbanding
     * Guild skills
     * Notices and expulsion history
     * Emblem selection
   * Pets
     * Capture slot machine
     * Egg hatching
     * Feeding
     * Status window and rename
     * Accessory equip/unequip
     * Performance actions
     * Emotes and talk bubbles
     * Feeding emotion reactions
     * Familiarity-gated client-side talk triggers
   * Companions
     * Homunculi
       * Status, skills, feeding, renaming, and deletion
       * Movement, combat, and skill commands
       * Default and custom USER_AI support
     * Mercenaries
       * Status and skill management
       * Movement, combat, and skill commands
       * Default and custom USER_AI support
     * Falcons
   * Friends
   * Parties
   * Whispers
   * Chat rooms
   * Character presentation
     * Item-specific weapon sprites
     * Mounts
     * Wedding sprites
     * Level 99 aura
 * UI
   * Overlapping windows with focus-to-front ordering and stable dragging
   * Basic information
   * Button bar
   * Multi-row shortcuts bar with classic key bindings
   * Console
   * Minimap with player, NPC, party, and guild markers
   * Items with vertical category tabs
   * Equipment
   * Option
     * Settings
   * Friends
   * Party & party settings
   * Guild management
   * Chat rooms
   * Stats
   * Skills with class-level vertical tabs
   * Homunculus and mercenary status and skill windows
   * Emote window
   * Cart Storage
   * Kafra Storage with vertical category tabs
   * Teleport skill modal
   * Warp skill modal
   * Cart appearance modal
   * Trade window
   * Vending windows
   * Card composition window
   * Show-equipment window
   * Alchemist crafting window
   * Blacksmith repair and refinement windows
   * Item pickup notifications
   * Item and skill tooltips
   * Status icons with roBrowser-sourced metadata
 * Emotes
 * Overlay text
   * FPS meter
   * Character names and HP/SP bars
   * Speech bubbles
 * Optional Lua character scripting
   * Player, enemy, nearby-player, companion, floor-item, and inventory state
   * Walking, stopping, attacking, looting, item use, chat, and targeted skills
   * Layout-independent physical keyboard input and text input
   * Lua-defined keyboard controls for movement, combat, looting, and skill target cycling
 * Tools
   * GRF packing and extraction
