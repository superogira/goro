# TODO

This file tracks intentionally ugly or temporary implementation choices, plus
large gameplay/rendering work that should not become invisible project
assumptions.

## Technical Debt

### Runtime And Portability

- **Lua 5.1 `.lub` execution**
  - Current state: RO `.lub` bytecode is parsed in `res/lub.go`, translated into
    GopherLua bytecode, then executed by GopherLua.
  - Ugly part: GopherLua requires a private `FunctionProto.stringConstants`
    field for `GETGLOBAL` and `SETGLOBAL`, so we currently set that field with
    `reflect` and `unsafe`.
  - Better fix: upstream or fork a small public API in GopherLua for externally
    built prototypes, or own a minimal Lua 5.1 bytecode runner for RO data files.
  - Also revisit whether the old fallback evaluator is still needed after enough
    real `.lub` files are covered by tests.

- **`nofakecgo` build tag requirement**
  - Current state: builds are expected to use `CGO_ENABLED=0 -tags nofakecgo`.
  - Ugly part: the tag exists to avoid duplicate fake-cgo symbol providers in
    the pure-Go GoGPU dependency stack.
  - Better fix: remove the tag requirement once the dependency stack no longer
    needs competing fake-cgo shims.

### Rendering

- **Dynamic world geometry**
  - Current state: static GND surfaces, lightmapped GND subdivisions, and RSM
    placements are retained `render.WorldMesh` objects. They are built once for a
    loaded map/model placement, uploaded into persistent GPU buffers, and reused.
  - Current state: actor/NPC/mob/item/effect sprite billboards use the dedicated
    shared-quad billboard pipeline with per-instance data instead of per-frame
    quad vertex slices.
  - Current state: shader-side camera projection and fog are used for retained
    meshes and billboards; render stats expose `world_mesh_commands`,
    `retained_world_meshes`, `world_billboards`, and remaining dynamic
    `world_vertices`.
  - Remaining dynamic geometry is intentional: animated water waves, warp/effect
    cylinders/rings, STR effects with animated per-corner coordinates, tile
    cursor geometry, and debug/fallback paths.
  - Better fix: move water and common effect primitives to specialized
    instanced/procedural GPU pipelines so `world_vertices` mostly measures only
    rare debug/fallback geometry.

- **Billboard depth and clipping**
  - Current state: sprites use 3D-ish billboard placement with custom depth
    handling to avoid being wrongly hidden by terrain/buildings.
  - Ugly part: RO sprites are not physically normal 3D quads; correct occlusion
    still needs policy knobs and reference-client comparison.
  - Better fix: document and implement the exact reference-client/OpenMidgard approach
    for sprite depth bias, top/head clipping, and object interaction.

- **Lighting model**
  - Current state: fog uses the reference client's camera defaults and near/far
    table values, applies the ramp at full strength, derives depth from the
    camera projection, and uses a non-additive color mix. Additive GND lightmap
    layers are explicitly exempt from fog so fog is applied once to the composed
    world color.
  - Ugly part: parts of the RSW/GND lighting pipeline are still approximations,
    especially the split base/lightmap rendering and per-map visual parity.
  - Better fix: collapse GND base, lightmap alpha, posterized light color, and
    fog into one shader path like the reference client, then add screenshot-style regression
    fixtures for representative outdoor, indoor, and dungeon maps.

### Gameplay State

- **Local death as held animation**
  - Current state: local player death uses the transient actor animation path,
    with the final death frame held until positive HP or map change.
  - Ugly part: reference client models death as persistent entity action/state, not as
    an expiring combat-style animation.
  - Better fix: split persistent actor state such as idle, walk, sit, and dead
    from transient overlays such as attack, hurt, and pickup, then clear dead
    state only on resurrection, respawn, or map transition.

### Data And Protocol

- **Fallback resource tables**
  - Current state: some NPC/monster/job/resource names still have hardcoded
    fallback maps.
  - Ugly part: these can hide `.lub` or DB parsing bugs and can become stale.
  - Better fix: rely on client data tables first, keep fallbacks only as explicit
    diagnostics or emergency compatibility.

- **Packet-version assumptions**
  - Current state: packet client date/profile defaults target the current rAthena
    setup.
  - Ugly part: several protocol choices are still date/profile-sensitive.
  - Better fix: centralize packetver negotiation/configuration and document which
    server builds are supported.

- **Real-data tests**
  - Current state: real RO data tests are gated by `GORO_DATA_DIR`.
  - Ugly part: CI cannot validate the most important compatibility paths without
    proprietary data.
  - Better fix: keep small synthetic fixtures for parsers and maintain a local
    real-data test checklist for releases.

## Effects

### Original-client job audit

Completed source audit following the first-job, Knight/Lord Knight, Crusader/Paladin,
Priest/High Priest, Monk, Wizard, Sage, Assassin/Rogue, and Soul Linker passes.
For each class: inspect every skill against dhxj and the 2008 HighPriest client,
cross-check server packets, implement discrepancies, review and fix twice,
validate, then commit and push. Existing support is not proof of visual parity.

Reference caveat from the Assassin Cross pass: HighPriest and the actual
`2008-09-10aSakexe.exe` are not interchangeable. HighPriest uses different
Soul Breaker/Meteor Assault poses and adds an Asura sound/quake to Soul Breaker.
Where they disagree, verify the actual Sakexe binary before choosing behaviour.

- [x] Hunter — traps, Detecting, Blitz Beat, Phantasmic Arrow, and utility/passive skills audited; reviewed twice and validated with OldRO assets.
- [x] Sniper — all four skills audited; procedural visuals, casting poses, hit routing, and sound timing reviewed twice and validated with OldRO assets.
- [x] Bard — songs/ensembles, Musical Strike, Frost Joke, Pang Voice, and utility/passive skills audited; reviewed twice and validated with OldRO assets.
- [x] Clown — all six 2008 skills audited; rays, sphere, link/body effects, Hermode cubes, and Tarot timing reviewed twice and validated with OldRO assets.
- [x] Dancer — all nine skills and shared ensembles audited; native dances, Slinging Arrow, Scream, and Wink reviewed twice and validated with OldRO assets.
- [x] Gypsy — all six shared skills and inherited dances audited; both reviews and actual female ACT checks passed, using the Clown/Dancer implementation.
- [x] Blacksmith — all 24 skills audited, including Full Adrenaline Rush; native buff visuals, packet ownership, Hammer Fall, and sound timing reviewed twice and validated with client assets.
- [x] Whitesmith — all five active skills audited; Cart Termination, buff poses/audio, and shared Overthrust afterimages reviewed twice and validated with client assets.
- [x] Alchemist — all 20 skills audited; bottle flights, Chemical Protection, homunculus casts, Demonstration, Berserk Pitcher, and Twilight Alchemy reviewed twice and validated with client assets.
- [x] Creator — all four active skills audited; potion bursts, Acid Demonstration, protection, and Cultivation reviewed twice and validated with client assets.
- [x] High Wizard — all six skills audited; Soul Drain, Amplification, Magic Crasher, Napalm Vulcan, Ganbantein, and Gravitation reviewed twice and validated with client assets.
- [x] Professor — all eight skills audited; particle trails, linking/body effects, ground units, casting, and sound timing reviewed twice and validated with client assets.
- [x] Assassin Cross — all five skills audited; slashes, poison particles, packet ownership, casting, poses, and sounds reviewed twice and validated with client assets.
- [x] Stalker — all four skills audited; casting spin/smoke, body flashes, Preserve, and Full Strip reviewed twice and validated with client assets.
- [x] Champion — all four added skills and shared sphere/Asura effects audited; native packet timing, poses, sounds, geometry, and body flashes reviewed twice and validated with client assets.
- [x] Taekwon — all 17 skills audited; kick poses/body effects, Run footprints, High Jump, Mild Wind, and sounds reviewed twice and validated with client assets.
- [x] Star Gladiator — all 18 skills audited; Warmth, Comfort, Hatred, Demon, and Fusion reviewed twice and validated with client assets.
- [x] Ninja — all 23 skills audited; native projectiles, particles, sprites, ground units, Mirror Image, and poses reviewed twice and validated with client assets.
- [x] Gunslinger — all 22 skills audited; native gun poses, packet ownership, coins, grenade units, sprites, and sound timing reviewed twice and validated with client assets.

Hunter reference checks: `dhxj` and 2008 `HighPriest` skill action/effect
routing, trap state changes, particle primitives, and rAthena trap packets.
Trap bursts are owned by look changes, Shock Wave loops on its unit, and
Remove Trap/Spring Trap resolve the trap position before removal. Added missing
procedural effects, original sound delays, Ankle Snare's one-shot model
animation, trap picking/chasing, and Talkie Box's bubble-only packet. Passives
(Beast Bane, Falconry Mastery, Steel Crow) have no invented activation effect;
the existing falcon option owns the bird. Full tests with client assets and a
normal build passed. Live visual comparison remains useful for fine tuning.

Sniper reference checks: both original clients agree on True Sight's layered
rays, Wind Walk's four strips and pale-green body flash, Sharp Shooting's
delayed rays and ACT 88 casting pose, and Falcon Assault's impact/quake timing.
The 2008 executable confirms the Korean sound filenames. Buff effects belong
to the packet source; secondary splash packets do not replay attacks. Falcon
Assault rotates only the rendered sprite during its burst, preserving the
server's facing afterwards. Full asset-backed tests and a normal build passed;
live comparison of these procedural visuals remains useful.

Bard reference checks: dhxj and 2008 HighPriest agree on the floating song
symbols, vertical ribbons, rotating bands, and Roki particles. Ground units own
these visuals; activation owns the single sound (Whistle and Dissonance are
silent). Musical Strike uses ACT 88 and the native arrow flight timing. Frost
Joke and Pang Voice use the finite shout pose; Pang Voice captures its target
position and camera-relative direction. Song/dance loops now use their correct
frame ranges and stop on Adaptation/status end. Musical Lesson has no activation
visual, and Encore uses the server's subsequent song notification. Full tests
with client assets and a normal build passed; live comparison remains useful.

Clown reference checks: dhxj and 2008 HighPriest agree on Arrow Vulcan's
36 delayed pink rays, white casting aura, ACT 88 release, and additional attack
replays. Moonlit uses a status-owned sphere on the caster, not ground tiles;
Marionette links two actors and applies the pink body glow until status end.
Hermode ground units own their blue cubes. Longing flashes the caster; Tarot
flashes its target and displays only the card selected by the server, at its
native height and fade timing, without duplicate sound. The later renewal
skill CG_SPECIALSINGER is outside this 2008 audit. Both review passes, full
tests with client assets, and a normal build passed; live comparison remains
useful for these procedural effects.

Dancer reference checks: dhxj and 2008 HighPriest agree on the five ground
visuals, including Slow Grace's green ribbons, Service's pink cross, and
Fortune's enlarged offset kiss. Ground units own the visuals; the skill packet
owns the sound. Slinging Arrow uses ACT 88 and native arrow timing, Scream
holds frame zero for 1200ms and sends only its caster's chat line, and Wink
uses a kiss emote plus the directional sprite shared with Pang Voice. Their
shared sprite primitive now follows the caster's lifetime while retaining the
captured target position. Dancing Lesson is passive; shared ensembles and
Adaptation/Encore reuse the Bard audit. Both reviews, full asset-backed tests,
and a normal build passed; live visual comparison remains useful.

Gypsy reference checks: dhxj and 2008 HighPriest use the same six CG skill
paths as Clown, without gender-specific effect variants; rAthena's pre-renewal
tree confirms that set and the inherited Dancer skills. The shared effects
and lifecycle checks pass. Actual Gypsy body assets were checked in all eight
directions: Arrow Vulcan releases with ACT 88, replays the normal whip attack
for subsequent hits, and the ensemble dance has its full 1,2,3,2 sequence.
No additional runtime code was needed. Both reviews and focused tests with
client assets passed; live comparison remains useful as for Clown/Dancer.

Blacksmith reference checks: dhxj and 2008 HighPriest agree on Adrenaline's
20 screen-space rays, four rising orbiting particles, and sounds at ticks 0/30.
Overthrust starts a fading ring and two sounds; its status tint and afterimages
were completed in the Whitesmith pass. Weapon Repair and Maximize Power use their complete STR duration and
native audio ticks; Weapon Perfection stays on the recipient. Hammer Fall uses
the attack pose while casting, releases in the skill pose, and places a fresh
STR on the ground with its sound at tick 30 and quake at tick 35. Greed uses
the pickup pose and caster sound. Forging/passive skills introduce no activation
visual; existing crafting-result packets own success/failure effects. rAthena's
party-buff notifications and roBrowser's explicitly simplified Adrenaline entry
were cross-checked. Both reviews, full asset-backed tests, and a normal build
passed; live comparison remains useful for the new procedural visuals.

Whitesmith reference checks: dhxj and 2008 HighPriest agree on Meltdown and
Cart Boost's standing release pose, caster ownership, STR assets, and sounds.
Maximum Power Thrust reuses the Overthrust ring; both statuses now produce the
native red tint and frozen, fading sprite copies (five-tick spacing, alpha 180
decreasing by eight per tick, no copies on dyed bodies). These reuse cached
billboards and are cleared on actor removal/map entry. Cart Termination starts
on its victim at release, including misses: at tick 30 it plays Fire Pillar's
impact sound and emits four groups of twelve pink-white light particles, with
native movement, slowdown, fade, and ground clipping. It adds no hit spark or
secondary-splash replay. Weapon Refinement already uses the item-selection UI
and server-owned refinement success/failure effects. Create Coin, Create
Nugget, and System Create are unused draft skills, disabled in rAthena's tree;
no behavior was invented for them. Both reviews, the full test suite with
client assets, and the normal build passed. Live visual comparison is still
useful, particularly for the new actor afterimages.

Alchemist reference checks: dhxj and 2008 HighPriest agree on the bottle
assets, 25-tick flight speed, arc, spin, afterimages, and potion-level colors.
Acid Terror throws at release; Potion Pitcher uses its success packet, while
separate server notifications own the healing visuals. Berserk Pitcher retains
its casting throw pose and applies the recipient's Berserk Potion effect,
including its red tint, growing pillar, rising streaks, and tick-18 sound/quake.
Chemical Protection now draws its eight blue-white rays on the recipient.
Homunculus utility skills use the pink Couple Casting rings. Demonstration's
unit owns its looping, ground-anchored sprite; ground notifications animate
the caster without duplicating the field. Twilight I–III use casting effects
573–575, scattering 320 ingredients around the local viewer as both originals
do, with incremental updates and at most three draw calls. Completion has
no additional flare or sound. ACT assets determine throw animation speed;
the old SetAction speed argument is unused in the 2008 executable. Pharmacy's
existing crafting UI/results and summon/homunculus entity lifecycles remain
server-owned. Passives and unused draft homunculus skills have no invented
activation effects. Both review passes, full asset-backed tests, and a normal
build passed. No live playtest was performed; visual comparison remains useful.

Creator reference checks: dhxj and 2008 HighPriest agree on the five-frame
ACT 40 casting sequence for Aid Condensed Potion and Acid Demonstration;
release preserves that pose. Potion bursts select red/yellow/white by level,
fall with three afterimages, and emit their expanding ground wave; the red
variant draws it twice. The activation sound uses the native fixed attenuation,
and separate server heal packets own healing. Acid Demonstration launches its
two bottles at cast ticks 5/10, then places four dark rings at the victim's
release position, with immediate sound, tick-5 quake, and per-hit flashes.
Textures preserve their original alpha and black outlines. Full Chemical
Protection reuses the recipient's chemical rays and blue-white body flash;
Cultivation adds only its white caster flash and skill pose, leaving plants to
the server. rAthena's tree and skill handlers confirm the four active skills;
Alchemy/Synthesis Potion are unused drafts, and inherited skills reuse the
Merchant/Alchemist audits. Both reviews, the full suite with client assets, and
a normal build passed. No live playtest was performed; visual comparison remains
useful, particularly for the new procedural bursts and rings.

High Wizard reference checks: dhxj and 2008 HighPriest agree on Soul Drain's
three curved particle trails, blue body flashes, and forced ACT 88 pose, which
preserves an existing walk. Mystical Amplification uses the server's buff state
to retain its alternating additive weapon pass; activation refreshes the effect
and plays the original Korean sound. The weapon alternation uses 60 Hz effect
ticks rather than render frames. Magic Crasher starts at release, shakes at tick
30 for 650 ms, flashes only the body additively, and retains per-hit flashes.
Napalm Vulcan plays its five delayed sounds; both originals call Hit2 after its
particle creation condition has passed, so no extra bursts were invented.
Ganbantein uses the native white ground rings and release sound. Gravitation's
ground unit owns its four stone/ice spikes and lifetime; ground notification
only animates/sounds the caster. Casting poses, aura colors, splash suppression,
and miss handling were checked against the original packet handlers and
rAthena. Professor HP Conversion's shared routing was corrected to effect 383
alone; its renderer remains for the Professor audit. Both reviews, the full
suite with client assets, and a normal build passed. No live playtest was
performed; visual comparison remains useful for the procedural effects.

Professor reference checks: dhxj and 2008 HighPriest agree on HP Conversion's
five groups of three green particle trails and delayed blue expansion of the
body. Soul Change now connects the actors with its two-color beam, flashes both
bodies, and plays the STR on both participants; the Korean filename was
incorrectly written as `사랑효과` instead of the original `사람효과`. Soul Burn
adds Magic Crasher's flash, sound, and quake on the server-selected recipient,
including the caster on failure. Its separate level-5 damage packet adds only
the generic hit flash. Mind Breaker follows the 2008 transient random tint and
tick-25 Provoke sound; later dhxj instead keeps the tint through the status.
Memorize flashes yellow and plays Suffragium; Double Casting flashes red with
an expanded body and centered amplification sound.
Wall of Fog uses the original translucent grey square walls and diagonals;
Spider Web uses the pulsing ground texture. Both are owned by server units,
without duplicate fields at cast notification. rAthena confirms the eight
active skills, packet outcomes, and unit lifetimes. Mage/Sage skills inherit
their earlier audits. Both reviews, full tests with client assets, and a normal
build passed. No live playtest was performed; visual comparison remains useful,
especially for the procedural trails and ground walls.

Assassin Cross reference checks: dhxj and HighPriest describe four delayed
Soul Breaker slashes and eight groups of four Meteor Assault slashes, with
acceleration, fade, fixed launch position, and a brief purple caster flash.
The actual 2008 Sakexe confirms normal-alpha pale-blue slashes (renderer at
`0x62a6e0`), weapon-dependent attack poses for both skills (`0x6847c0`), and
no extra Asura sound/quake in Soul Breaker's release (`0x686972`). These last
two behaviours differ from HighPriest. Soul Breaker's spatial sound starts
at tick 20; Meteor Assault sounds immediately and only its successful
no-damage packet starts the burst. Damage packets add the ordinary hit flash;
secondary splash packets do not replay effects. Enchant Deadly Poison restores
the small rising particles, alternating purple body flash, and six Enchant
Poison sounds at ticks 0/5/11/18/26/40. Create Deadly Poison reuses crafting
success/failure effects; rAthena owns production, self-damage on failure, and
the poison status. Advanced Katar Mastery is passive. Both reviews, full tests
with client assets, and a normal build passed. No live visual comparison was
performed; the procedural geometry still benefits from playtesting.

Stalker reference checks: dhxj and HighPriest agree on the four skill
notifications, particle primitives, and body flashes. The actual 2008 Sakexe's
release table (`0x6847c0`) and casting table (`0x685050`) confirm Chase Walk's
standing spin followed by the skill pose, Preserve's standing purple cast,
and the ordinary casting/release poses for Reject Sword and Full Strip.
Chase Walk plays the native smoke ACT once at 1.2 frames per tick, fixed at
the release position on both activation and deactivation. Its casting spin
uses the native acceleration and stops on release/interruption without
changing authoritative facing. Reject Sword and Preserve belong to the caster;
the former adds its alternating grey expanded body, while the latter reuses
the Guard mesh with its larger radius, rotating initial orientation, white
flash, and native sound. Full Strip emits sixteen shields and sixteen swords
around the victim, with randomized delays, additive red tint, native travel
speed, and four sounds. These are successful no-damage notifications; damage
packets do not replay the buff/strip effects. rAthena owns the stealth toggle,
reflect triggers, strip result, and status lifetime. Both reviews, full tests
with OldRO assets, and a normal build passed. No live visual comparison was
performed; the new procedural effects still benefit from playtesting.

Champion reference checks: dhxj's `Am_Skill`, `Zc_useskill_ack`, effect
primitives, HighPriest, and the actual 2008 Sakexe agree on the four added
skills' routing. Sakexe confirms Palm Strike's skill frames 3–4 and Tiger
Fist's held frame 4 (`0x578242`, `0x5782d2`), unlike HighPriest's ordinary
attack poses. Palm Strike starts its red bouncing trails on the victim;
Tiger Fist starts its blue/yellow rays and yellow rings on the caster.
Chain Crush emits 32 flat streaks aimed at the victim, with sound and quake
at tick 44. Secondary splash packets retain damage feedback without replaying
the attack. Hit markers remain separate from release effects. Zen launches
its yellow wind during casting, including a zero-time casting notification,
holds skill frame zero, and finishes standing; sphere counts come only from
server packets. The inherited sphere skills use the same casting pose.
The shared Asura implementation now uses six glyphs at 20-tick intervals,
ten fading layers, native ring timing, one casting sound, and separate captured
origins for the two glyph groups. Sakexe's `BeginAsura` (`0x611af0`) and
`RenderAsuraCasting` (`0x61f670`) confirm the dark Champion glyph variant;
HighPriest uses a later blue additive variant. This pass also corrects the
shared Combo Finish sound and yellow combo rings. Both reviews, the full
test suite with OldRO assets, and a normal build passed. No live visual
comparison was performed; these procedural effects still need playtesting.

Shared audit correction: dhxj, HighPriest, and the actual 2008 Sakexe's
no-damage (`0x59c930`) and ground (`0x578850`) handlers read the release
action table for animation only. Removed the extra flares/sounds previously
added to buff and crafting completions and Summon Marine Sphere's release flare.
Concentration's AGI/DEX overlay belongs to casting and follows its duration;
completion plays only the Concentration STR. Damage packets retain their explicit
begin effects. Ordinary casting auras now follow the packet's element,
matching the final override in `GetSkillActionInfo2` (`0x6853a9`): water,
earth, fire, wind, poison, and holy select their colored variants; ghost
remains neutral. Skill-specific auras still take precedence. Both reviews,
full tests with OldRO assets, and a normal build passed; no live playtest was
performed for these corrections.

Taekwon reference checks: dhxj's packet handlers, action table, body transforms,
and procedural primitives were cross-checked with HighPriest, the actual
2008 Sakexe, and rAthena's skill/status handlers. Ready poses come from skill
notifications, not stance toggles. Heel Drop, Roundhouse, and Counter use their
five selected ACT motions; secondary damage does not replay the release or
inherit the primary hit's extra delay. Tornado Kick belongs to the no-damage
notification. Run observes the server's status/no-damage ordering and leaves
alternating footprints behind after stopping. High Jump separates takeoff,
dust, and landing without moving the authoritative actor. Mild Wind uses a
captured tornado plus one following white elemental glyph, reusing the Asura
primitive. Passive recovery/party power remain server-owned; Mission retains
its existing packet/UI handling and adds the original sound. Sakexe confirms
the Korean Taekwon sound filenames where dhxj uses renamed English files.
Both reviews, full tests with OldRO assets, and a normal build passed. Tests
also check update-rate independence and male/female Taekwon-family ACT frames.
No live visual comparison was performed. Body rotations use Goro's composed
sprite layers rather than reproducing the original per-accessory pivot code;
these transforms and the new procedural visuals need playtesting.

Star Gladiator reference checks: dhxj, HighPriest, and the actual 2008
Sakexe agree on Warmth's persistent red lightning/body glow plus release
particles, Comfort's eight blue rings and body overlays, and Hatred's targeted
particles and registration sound. These replace roBrowser's mismatched skill
mappings. Warmth and Comfort hold skill frames three and two respectively;
Feeling, Hatred, and Fusion use BlueCasting. Warmth follows server option/status
updates and restores on actor entry; no-damage notifications own the release
particles and delayed sounds. The shared Hated effect is silent; Kaahi retains
its own skill sound. Demon uses its learned level to choose the native vision
mask and black rings, tolerates either status/skill-list packet order, and
cleans up when the status ends. The opaque surround extends to modern viewport
sizes. Fusion uses the server's flying option to select the alternate ACT,
hover while standing/moving, rise while sitting, and settle during actions or
when the option ends. Its release plays only the original sound and quake.
Feeling/Hatred registration retain their existing UI; Anger, Blessings, Friend,
and Knowledge remain server-owned. Both reviews, the full suite with OldRO
assets, and a normal build passed. No live visual comparison was performed;
the procedural rings, vision mask, and composed-sprite flight need playtesting.

Ninja reference checks: dhxj Session2/GameActorMsgHandler/GameActor,
RagEffect/RagEffect2/RagEffectPrim and Skill.cpp, HighPriest, the actual
2008 Sakexe, roBrowser's mappings, and rAthena's Ninja skill implementations.
All 23 skills were checked. Throwing Mastery and Ninpou remain server-owned;
Shadow Leap, Cicada, and Soul keep their server movement/status behavior.
The 2008 executable confirms the two ACT-96 release sequences and casting
poses. Throw Shuriken, Kunai, Huuma, and Zeny now launch one native arcing
projectile per notification, with their centered throw sound. The Huuma
texture matches 2008. Fire Petal emits twenty independently drifting sakura
sprites, two BASH3D ray groups, and four delayed sounds; its impact uses FireHit.
Ice Spear reuses the native Earth Spike geometry with ice texture, captured
position, sound, quake, and ColdHit. Wind Blade follows the target with the
packet hit count; Kamaitachi emits five vertical wind blades. Both retain
native trail histories. Shadow Slash, Vanishing Slash, and Final Strike use
the native sprite actions, timing, offsets, and blending. Fire Dragon, Ice
Meteor, and Lightning Strike belong to no-damage release packets, with the
original target/caster ownership; Lightning Strike uses its looping SPR.
Tatami, Fire Formation, and Hidden Water belong to server skill units rather
than ground-cast notifications. Fire Formation overlaps its short animations
and has the original one-in-eight cell-entry sound. Mirror Image supplies
blue tint and frozen afterimages, including on dyed bodies, and cleans up on
status end. Existing shared rendering handles rings, spikes, throws, and rays.
Both reviews, full tests with OldRO assets, and a normal build passed. No live
visual comparison was performed; the new procedural effects need playtesting.

Gunslinger reference checks: dhxj Session2/GameActor/GameModePacket,
RagEffect/Effect_SPR/CHEMICAL3/CHOOKGI/RG_COIN2, the HighPriest decompilation,
the actual 2008 Sakexe effect cases and release-pose table, roBrowser mappings,
and rAthena's Gunslinger implementations and ground-cast notification path.
All 22 skills were checked, including the three passives (no independent
activation visual). Attacks launch their special visual once per damage
notification; Disarm, Desperado, and buffs use the separate successful
no-damage notification. Tracking no longer emits a spurious casting STR.
Added red/blue buff flashes, Increasing Accuracy streaks, Piercing Shot rays,
and Disarm weapon particles using existing native primitive renderers.
Corrected sprite direction, playback speed, sound delays, and Triple Action's
30-tick sprite delay. Flip Coin and Cracker use centered sounds. Ground Drift's
five grenade types belong to their server units and overlap 23-tick sprite
plays every 16 ticks. Coins follow the reported count, show at most five gold
orbs (larger orbs represent counts above five), and reset on count changes.
Their cosmetic wandering catches up for at most two seconds after a long
pause. Native gun release/casting poses reuse the actor animation chain,
including the randomized forward/reverse buff motion. The former NPC Power Up
alias for EF_AGIUP was corrected while preserving the NPC skill's existing
sound. Both reviews, the full test suite with OldRO assets, and a normal build
passed. No live visual comparison was performed; the new visuals need
playtesting, as do the other procedural effects in this audit.

Previous catalog import source: roBrowser's `src/DB/Effects/EffectConst.js`,
`src/DB/Effects/EffectTable.js`, `src/DB/Skills/SkillEffect.js`,
`src/DB/Skills/SkillUnit.js`, and `src/DB/Items/ItemEffect.js`.

### Status

- Reference active numeric `EffectTable.js` entries: `607`.
- Reference numeric effect constants: `1147`.
- Goro effect catalog entries: `750`.
- Catalog size relative to all numeric constants: about `65.4%`.
- The catalog is not a strict subset of the active table: it also contains
  inactive, later-client, and synthetic routed effects. An exact active-table
  intersection report is still part of the backlog below.
- Coverage only counts world/effect IDs handled through `worldEffectSpecForID`
  and direct effect trigger mappings. It does not count UI rendering, damage
  number rendering, cursor drawing, actor sprite animation, or map logic that is
  not modeled as an effect ID.

### Current Shape

- Effects are primarily data-table driven through `skillEffectSpecs`,
  `worldEffectSpecs`, and parser-backed tests against roBrowser's effect table.
- Renderer support exists for STR, CYLINDER, 2D billboards, 3D billboards,
  SPR/ACT attachments, sprite-backed 3D effects, duplicate timing, delays,
  basic alpha/fade, color tint, size interpolation, and source-to-target motion.
- Special Go behavior is still allowed for effects that are not purely
  declarative in roBrowser data, such as camera shake, map transitions, or UI
  choices produced by a skill.
- RSW-placed map effects and map-wide weather are tracked separately in
  [docs/map-effects-weather-todo.md](docs/map-effects-weather-todo.md).

### Recently Covered Families

- Swordman first-job effects: Bash, Provoke, Magnum Break, Endure.
- Mage first-job effects: Napalm Beat, Soul Strike, Cold Bolt, Frost Diver,
  Stone Curse, Fire Ball, Fire Wall, Fire Bolt, Lightning Bolt, Thunder Storm,
  Sight, Safety Wall, and Energy Coat routing/effects where currently supported.
- Acolyte first-job effects: Heal, Increase/Decrease Agility, Aqua Benedicta,
  Signum Crucis, Angelus, Blessing, Cure, Ruwach, Pneuma, Teleport, Warp Portal,
  and Holy Light.
- Thief first-job effects: Steal, Hiding, Envenom, Detoxify, Sprinkle Sand,
  Throw Stone.
- Archer first-job effects: Improve Concentration, Double Strafe, Arrow Shower,
  arrow projectile and ammo handling.
- Merchant first-job effects: Mammonite, Cart Revolution, Crazy Uproar,
  cart state/appearance, vending bubbles and shop interaction.
- Ninja expanded-class support: normal/baby jobs, full skill tree and level
  metadata, roBrowser skill actions/effects, and Ninja status routing.
- Gunslinger expanded-class support: normal/baby jobs, full skill tree and
  level metadata, native gun poses, coins, grenade units, and skill effects.
- Taekwon-family support: normal/baby Taekwon, Star Gladiator, and Soul Linker
  jobs, skill trees, stance/status behavior, mission packets, and ranking UI.
- Common effects: hit feedback, potion/food families, Heal/recovery feedback,
  base/job level-up, teleport/portal, refine/pharmacy success/failure.

### Priority Backlog

- [ ] Improve STR parity: blend modes, alpha edge behavior, texture frame
  selection, UV animation, layer timing, rotation/scale interpolation, source
  offsets, and subtractive/additive modes.
- [ ] Improve CYLINDER parity: partial circles, repeat texture axes, fixed
  perspective versus world-space orientation, rotation deltas, duplicate
  instances, randomized fields, and exact alpha behavior.
- [ ] Complete reference `wav` behavior: delayed sounds, randomized sounds,
  positional sounds, and no inferred sounds when roBrowser declares none.
- [ ] Finish generic routing for `SkillEffect.js`, `SkillUnit.js`, and
  `ItemEffect.js` so adding new effects is mostly data import instead of custom
  Go code.
- [ ] Add persistent attached lifecycle handling for buffs/status loops:
  follow actors while active, detach only when the reference client does, and
  clear on vanish/death/status removal.
- [ ] Add a debug command or CLI flag that spawns an effect ID at the player for
  fast visual comparison.
- [ ] Generate an `effects_coverage.json` style report with support levels:
  `none`, `sound-only`, `partial`, `close`, and `parity`.

### Gameplay-Visible Gaps

- [ ] Finish live visual/timing parity for hit and combat feedback. Catalog
  routing now covers `EF_HIT1` through `EF_HIT6`, elemental/poison variants,
  critical-related effects, and blocked/zero-damage feedback, but source-level
  coverage is not the same as verified visual parity.
- [ ] Finish ground-skill lifecycle and visual parity. Routing exists for
  Sanctuary, Magnus, Quagmire, traps, Bard/Dancer songs, Volcano, Deluge,
  Violent Gale, Land Protector, Spider Web, Basilica, Suiton, and Epiclesis;
  remaining work is exact geometry, timing, server update/removal behavior, and
  live comparison.
- [ ] Status loops: poison, curse, blind, silence, stun, sleep, confusion,
  frozen, stone curse, hiding/cloaking/invisible states, spirit spheres, aura
  loops, and actor-attached buff/debuff decorations.
- [ ] Second-job pre-renewal sweep: Knight/Crusader, Wizard/Sage,
  Priest/Monk, Assassin/Rogue, Hunter/Bard/Dancer, Blacksmith/Alchemist.
- [ ] Remaining expanded and later content: Super Novice, transcendent classes,
  renewal third jobs, and modern fourth-job effect IDs.
