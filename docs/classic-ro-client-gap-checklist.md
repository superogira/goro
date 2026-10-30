# classic-ro-client Gap Checklist

Updated 2026-10-22 against the current Goro code.

This checklist records features and smaller behaviors present in
`~/src/classic-ro-client` that are absent or incomplete in Goro.

The reference client's `Features.md` says it was validated with rAthena
packet version `20120307`, while Goro targets `20080910`. Items in the main
checklist are applicable to the 2008 client or are packet-independent client
features. Anything that may depend on later client behavior is kept in a
separate validation section and must not be implemented blindly.

## Gameplay and protocol compatibility

### Stealth

- [x] Use server actor options for Hide, Cloaking, Invisible, and Chase Walk, independently of status-icon timers.
- [x] Apply the 2008 hidden-viewer rendering rules to local and remote actors.
- [x] Apply movement and action restrictions, including Tunnel Drive, the six skills usable while Hiding, and Chase Walk's own toggle.
- [x] Exclude hidden actors from picking and targeting, and cancel queued actions when visibility changes.
- [x] Suppress hidden actors' names, emblems, auras, carts, and falcons.
- [x] Add focused rendering and packet tests for self, party, detection, GM, ordinary viewers, PvP, and WoE.
- [ ] Compare the full flow visually against a running 2008 client and server.

The 2008 executable differs from classic-ro-client:
Hiding keeps a shadow for self/GM, Cloaking hides even self and party bodies,
Clairvoyance reveals a black body without a shadow, and Invisible remains
unseen. The later translucent party view is deliberately not imported.
Seeing a silhouette does not permit selecting a hidden target.

Verified against `~/src/ro-client-re`'s `2008-09-10aSakexe.exe`:
`CGameActor::SetAttrState` at `0x00572b80`, `CPc::Render` at `0x005e4920`,
`CSession::IsMasterAid` at `0x006711a0`, and `CPlayer::SendMsg` at `0x005ee950`.
The Ninja skill exceptions and server option combinations were also checked
against `~/src/eathena/src/map/status.c` and rAthena. No new packets or
duplicated stealth state are needed.

### Legacy mail

- [x] Implement mailbox refresh and inbox listing.
- [x] Implement reading mail.
- [x] Implement composing and sending mail.
- [x] Implement item and Zeny attachments.
- [x] Implement taking attachments.
- [x] Implement deleting and returning mail.
- [x] Implement mailbox and read-mail windows using Goro's existing window components.
- [x] Handle the legacy 2008 mail packets `0x023F`, `0x0241`, `0x0243`, `0x0244`, `0x0246`, `0x0247`, `0x0248`, and `0x0273`.
- [x] Add automated coverage for empty/full mailboxes, attachments, server errors, and pending acknowledgements.
- [ ] Complete live mail acceptance, including reconnects, after resolving the server issues noted below.

This means the legacy mail system, not a newer RODEX-only implementation. See
[the per-feature packet audit](packet-coverage-20080910-per-feature.md) and
[mail server compatibility and live checks](rathena-setup.md#legacy-mail-compatibility).

### Skill-specific client flows

- [x] Implement the Sage Auto Spell choice list and send selection packet `0x01CE`.
- [x] Turn the Star Gladiator place/Feel request into the original confirmation flow instead of only logging it, then send `0x0254`.
- [x] Implement the original `/doridori` client behavior and packet `0x01E7`.
- [x] Recognize the original Super Novice public-chat prayer and send `0x01ED`; the server checks EXP and grants Fury.
- [x] Implement Token of Siegfried self-revival through the 2008 auto-revive request (`0x0292`).
- [ ] Complete Auto Spell prompt-to-reply integration coverage for confirmation, cancellation, and empty lists. Feel confirmation, Super Novice prayer, and revival already test outbound requests.

### Storage password

- [x] Parse the storage-password prompt and result states.
- [x] Implement setting a new storage password.
- [x] Implement entering an existing storage password.
- [x] Handle confirmation mismatch, failure counts, and lockout/penalty state.
- [x] Ensure storage does not open before successful authentication when the server requires a password.
- [x] Implement the missing `0x023B` request flow and verify its 2008 wire layout against rAthena's packet database.
- [ ] Run the flow end to end against a server implementation. The local rAthena accepts the packet layout but still leaves `clif_parse_StoragePassword` as a stub.

### Rankings and fame feedback

- [x] Implement `/blacksmith` top-ten ranking request and response (`0x0217`).
- [x] Implement `/alchemist` top-ten ranking request and response (`0x0218`).
- [x] Format ranking entries consistently in the console.
- [x] Handle Blacksmith and Alchemist fame-point gain notifications.
- [x] Preserve the existing TaeKwon mission and ranking implementation.
- [ ] Investigate the `0x0237` killer ranking separately before treating it as a 2008 parity requirement.

## Client UI and quality-of-life

### Window state persistence

- [x] Assign stable persistence keys to the supported game-window preferences in `goro.ini`.
- [ ] Finish saving and restoring window positions between application sessions. Console, shortcut bars, and Stats are done; other ordinary windows currently retain positions only within a session.
- [x] Save and restore open/closed state where appropriate.
- [ ] Complete original-style window collapse/minimize behavior where needed. Basic Info and its menu already support compact/collapsed modes.
- [x] Save and restore Basic Info/menu compact and collapsed state.
- [x] Save the shortcut bar's visible row count and the skill window's tree/list mode.
- [ ] Decide whether to persist Battle Mode; it currently starts disabled on login.
- [x] Clamp restored windows to the current screen after a resolution or scale change.
- [x] Do not persist transient dialogs, context menus, tooltips, or server-owned modal state.
- [x] Save and restore camera zoom.
- [ ] Complete live checks for logout, character change, application restart, resolution change, and fractional scaling. Restore/relogin, drag saving, reopening, and screen clamping already have automated coverage.

These preferences are saved in the `[ui]` section of `goro.ini`, alongside the
existing native-window size and fullscreen settings. Game-window sizes follow
their selected layout modes.

### Keyboard shortcuts

- [x] Fixed classic window shortcuts: Alt+E inventory, Alt+Q equipment, Alt+S skills, Alt+A stats, Alt+Z party, Alt+H friends, Alt+W cart, Alt+C chat-room creation, Alt+V basic-info compact mode, Alt+J pet, Alt+R homunculus, Ctrl+R mercenary, and Alt+O settings. These use physical keys, like the existing shortcuts.
- [ ] Add a key-rebinding window instead of relying only on the fixed classic F1-F9 and Battle Mode mappings.
- [ ] Persist configurable physical-key bindings. Existing fixed shortcuts already use physical keys, including on AZERTY.
- [x] Add the original Alt+M shortcut-list window for chat-command bindings.
- [x] Support editing and clearing all Alt+1 through Alt+0 command slots, saved locally. The View button reuses the emote picker; shortcuts use physical keys (including AZERTY) and the console's command handling.
- [x] Classic Battle Mode: `/bm` or `/battlemode` toggles physical Z-., Q-O, and A-L shortcut rows. With it off (the login default), typing goes directly to chat; with it on, Enter or Space opens chat temporarily. F1-F9 work while chatting, and F12 cycles the selected bar independently of the +/− row controls.
- [x] Keep server-side item/skill hotkey slots distinct from client-side physical key bindings.

### Missing-map recovery

- [x] Preserve and surface the missing/unreadable GAT error instead of silently returning from map initialization.
- [x] Close the failed map connection and show the existing error alert on login, with its usual background and cursor fallbacks.
- [x] Display the missing map name and a useful explanation.
- [ ] Offer direct return to character selection.
- [ ] Optionally offer a recovery warp to Prontera when the connected server permits it.
- [x] Avoid leaving the player connected on a black, unusable map.
- [x] Test the known `new_1-1`-missing scenario.

This is a safety improvement from classic-ro-client rather than strict
original-client behavior, but it directly addresses a failure already seen in
Goro.

The implementation deliberately returns to login and suppresses automatic
reconnection/reselection. It reuses the existing alert and adds no recovery
phase, cursor, text wrapper, or map-server handshake. Direct character selection
and the optional Prontera warp are outside this minimal scope.

References checked in the local clones: classic-ro-client's
`map_missing_window.rs`, robr's `MapRenderer.js`/`MapEngine.js`, open-midgard's
`GameMode.cpp`/`GameModePacket.cpp`, and rAthena's `clif.cpp`. The richer recovery
flows keep the map session alive; Goro simply closes it. Tests cover missing and
invalid maps, entry/warp redirects, the error dialog, and explicit login retry.

### Minimap details

- [ ] Verify guide-direction marker parity against the 2008 client using the existing server compass support.
- [x] Party-member minimap markers already exist.
- [x] Same-map guild-member markers already exist.
- [x] Server compass markers already exist.

Party-name hover belongs to the 2007 world-map feature, not to the small HUD
minimap. The latter only needs the existing coloured party markers for 2008
parity.

### World map (2008)

- [x] Load the original `worldmap.bmp` and `mapPosTable.txt` from client data.
- [x] Open with Ctrl + the physical key left of 1, and close with the same shortcut or Escape.
- [x] Mark the current map with a star and highlight maps containing online party members.
- [x] Show a hovered map's minimap and party-member names; show known same-map positions in the preview.
- [x] Toggle map names using the title-bar magnifying glass.
- [x] Add resource, shortcut, scaling, input, lifecycle, and idle-redraw regression tests.
- [ ] Verify the full interaction visually in game, including party members on other maps.

The 2008-09-10 Sakray executable references both original assets, and OldRO
contains them.

### Graphics options and small rendering details

- [x] Expose runtime fog and aura controls: `/fog` toggles map fog and `/aura` simplifies level-99 auras; both save to `goro.ini`.
- [x] Expose `/effect` to hide optional visual effects while retaining essential feedback and portal markers; keep `/mineffect` for simplified effects.
- [ ] Implement Hallucination's full-screen ripple/distortion effect.
- [ ] Ensure screen distortion affects the scene without corrupting UI input or allocating render targets every frame.
- [ ] Consider smoothly fading the existing Taekwon Soul Link/Eske night tint instead of switching it abruptly.
- [ ] Verify the lighting behavior against the 2008 client before describing it as a general day/night cycle.

## Long-tail effect coverage

The job-by-job passes and effect catalog tests now cover broad skill/effect
support. The remaining work is identifying and validating actual 2008 gaps;
registry counts alone are not visual-completion percentages.

- [ ] Produce a machine-readable list of effect IDs used by the 2008 data/server profile that have no `db.EffectSpecs` entry.
- [ ] Separate truly used 2008 effects from later numeric constants.
- [ ] Compare remaining missing effects with the original client, robr, and classic-ro-client before implementing them.
- [x] Prefer generic SPR/ACT and STR players where the original data fully describes the effect.
- [x] Add custom behavior only when the reference client requires it.
- [x] Add structural and rendering regression tests for imported effect families.
- [x] Keep unsupported effects explicit; do not guess visual behavior.

classic-ro-client claims handling for every effect ID from 1 through 1050 by
combining generic players and custom effect families. Goro intentionally leaves
an effect unsupported when its reference behavior has not been established.

## Version-sensitive items requiring validation

Unchecked entries still require validation before implementation. First establish
that the feature belongs to the 2008 client and that OldRO contains the required
assets/tables. Checked entries record established scope decisions.

- [ ] Validate skill/global cooldown packets for `20080910` before adding cooldown gating or shortcut overlays.
- [ ] Validate whether the status-icon clock-wedge display is appropriate for 2008; Goro already has tooltips and a duration bar.
- [x] Exclude party-booking packets `0x0802` and `0x0806`: Hercules and roBrowser date them to 2009-12-22; rAthena's older-profile inclusion is caused by a `2009122` version-guard typo.
- [ ] Validate killer ranking `0x0237` independently.
- [x] Keep newer RODEX and later standalone cash-shop behavior outside the 2008 scope. Legacy NPC cash-point shops are confirmed in eAthena's February 2008 history and the 2008 Sakexe. The same binary contains memorial-dungeon information-window handlers, although eAthena implemented instancing in June 2009. Legacy auctions also predate the target. Track those optional systems in [the updated packet audit](packet-coverage-20080910.md), without assuming when particular server content was released.
- [x] Do not treat classic-ro-client's JSON companion AI configuration window as a direct parity requirement; Goro intentionally supports Gravity-style `AI.lua`/`AI_M.lua` instead.

## Verified existing Goro behavior

These reference-client features were checked and should not be re-added as
missing work.

- [x] Party-member minimap markers.
- [x] Same-map guild-member minimap markers.
- [x] Server compass markers.
- [x] Status-icon tooltips and remaining-duration presentation.
- [x] Item information illustrations, full card illustrations, and card-slot/card composition support.
- [x] Talkie Box and Graffiti text prompts and outbound talkbox skill packet.
- [x] Positional actor, effect, and RSW audio.
- [x] Party creation, invitation by name, settings, and member actions.
- [x] Item-pickup notification.
- [x] TaeKwon mission and ranking support.
- [x] Cast bars, projectiles, camera shake, MVP effect, and level-up effects.
- [x] Loading transition with an initial black cover and fade.

## Reference locations

- `~/src/classic-ro-client/Features.md`
- `~/src/classic-ro-client/docs/TODO.md`
- `~/src/classic-ro-client/lib/ui-component/src/game/`
- [Goro 2008 feature packet audit](packet-coverage-20080910-per-feature.md)
- [Goro detailed 2008 packet audit](packet-coverage-20080910.md)
