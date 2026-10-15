# 2008 Packet Coverage

Target client date: `20080910`. Updated **2026-10-15**, against Goro
`main` at `d9ca64e99296a31bb9331df60b44d7f8e1664c89`.
This includes the merged Monk and Super Novice prayer work; unmerged branches
are not counted.

## Summary

| Metric | Previous published summary | Current audit |
|---|---:|---:|
| Effective rAthena map table opcodes | 603 | 603 |
| Opcodes referenced in Goro network source | 232 / 603 (38.5%) | **301 / 603 (49.9%)** |
| Accepted client request opcodes referenced in Goro | 104 / 177 (58.8%) | **148 / 177 (83.6%)** |
| Accepted request opcodes actually emitted for the 2008 profile | Not measured | **132 / 177 (74.6%)** |
| Player request handler families with a sender, after the exclusions below | Not measured | **132 / 152 (86.8%)** |

The first two coverage rows preserve the old reference-count metric. They are
**not feature-completion percentages**: length-table entries, unused constants,
and opcodes used for another date also count as references. The old detailed
list had accumulated 633 rows, including entries outside the 603-opcode server
table; the regenerated table below uses the actual effective table again.

The 177 request registrations break down as follows:

- **132** have a sender for this profile.
- **6** are alternatives to an action already available: `0x0072`, `0x009B`,
  `0x009F`, `0x00FF`, `0x012F`, `0x0437`. This does not claim Goro emits those
  particular layouts; for example, party replies use `0x02C7`.
- **20** are remaining player request handlers, listed by feature below.
- **17** are GM/admin opcodes, representing 15 handlers.
- **2** are post-2008 party-booking requests accidentally admitted by the
  rAthena version guard.

For the 132/152 planning metric, group registrations by rAthena handler
(170 handlers), remove 15 admin handlers, two party-booking handlers, and the
older party-reply handler whose action is covered by the newer reply. This
leaves 152 player handler families, 132 covered and 20 without senders. It
still counts optional features and utilities, and says nothing about the
quality of their visual presentation or their incoming packets.

See [the per-feature checklist](packet-coverage-20080910-per-feature.md) for
work estimates. The remaining protocol work is about a dozen feature areas,
not one feature per unreferenced opcode. Rendering, effects, skill behavior and UI parity
need separate estimates even when their packets are supported.

## Sources and method

Local checkouts used:

- Goro-compatible rAthena `7f30a25e62411b1e1c17c4f21eb151e96eeaf37a`:
  `src/map/clif_packetdb.hpp`, `packets.hpp`, `packets_struct.hpp`, and
  `src/common/packets.hpp`.
- eAthena `0cb07145e`, including its Git history and September 2008 snapshot
  `dcccf4c9f4e683adf9de8a9901cac7dbfa165edf`.
- Hercules `a455f04df21f092bff3d7fce5726a7ad0c518d3c`:
  `src/map/packets.h` for independent version guards.
- roBrowser `aedce94228b5526140391214cb0d74ed08ca346c`:
  `src/Network/PacketVersions.js` and `PacketRegister.js` for version and name
  cross-checks, not as a feature-completion denominator.
- `ro-client-re`'s exported packet dispatcher for `2008-09-10aSakexe.exe`:
  `ghidra/data/sakexe-pollnetworkstatus-packet-cases.tsv` and
  `dhxj-packet-switch-aligned.tsv`, for the specific historical checks below.

The server headers were preprocessed with `PACKETVER=20080910`,
`PACKETVER_MAIN_NUM=20080910`, and `PACKETVER_SAK_NUM=20080910`. Resolve the
header/enum aliases, retain the last `packetdb_addpacket` per opcode, and
intersect with integer references in non-test `network/*.go` source. There are
886 declarations, 603 winning opcodes, 283 overwritten declarations, and zero
unresolved aliases. Included headers are not needed for evaluating these
conditional declarations; struct sizes remain symbolic in the table.

The sender audit additionally exercised the public client send methods with
representative arguments at client date 20080910, capturing their bytes in a
temporary Go overlay instead of opening connections. Map entry and keepalive,
which enqueue directly, were checked through their builders and call sites.
Empty-name party invitations, ranking kinds, and both boolean choices were
included. This establishes request routing, not full layout or server acceptance
validation. Parser/game-handler call sites were inspected for the incoming gaps.
No game code was changed and no packets were sent to a live server for this audit.

A map DB entry without a parse handler is a **length registration**, not proof
of an active server-to-client packet. It can be obsolete, unused, or even a
login/client request. Conversely, server messages such as `0x0215` are sent
without appearing in this table. A single S->C completion percentage from this
table would therefore be misleading.

Table statuses:

- `sender`: emitted by an active request path at the target date.
- `alias-covered`: the action is available through another accepted packet.
- `missing`: player request without a sender.
- `admin gap`: missing GM/admin request; chat atcommands can cover many uses.
- `post-2008`: not a requirement for the target client.
- `referenced`: non-request entry mentioned in network code, possibly only in
  the length table; does not imply a parser or game behavior.
- `untracked`: no network-code reference; does not by itself imply missing work.

## Historical scope checks

| Feature | Evidence | Scope for this audit |
|---|---|---|
| NPC cash-point shop, `0x0287`-`0x0289` | eAthena added the implementation in `931f73cdd` and the sample iRO-style Kafra shop in `c82fd53e6`, both **2008-02-28**. The September 2008 snapshot contains it. The target Sakexe dispatches `0x0287` to `0x005bf540` and `0x0289` to `0x005bf690`. | Confirmed old NPC shop protocol; optional if the server does not offer it. This is not the later standalone cash-shop interface. |
| Memorial-dungeon window, `0x02CB`-`0x02CE` | eAthena implemented instancing in `ee70fdc7d` on **2009-06-20**, so it was absent from that server's September 2008 implementation. However, the target Sakexe already dispatches these four packets to `0x005c1990`, `0x005c1a90`, `0x005c1b20`, and `0x005c1c50`. | The 2008 client has the information-window handlers; this does not date the release of individual dungeons. Optional UI/compatibility work when using instances on our current server. The `0x02CF` destroy command still needs a precise original-client send-path check. |
| Party booking, `0x0802` / `0x0806` | rAthena's guard says `PACKETVER >= 2009122` (one digit missing). Hercules and roBrowser place these requests at **2009-12-22**. | Exclude from the 2008 backlog. |
| Auctions | eAthena and Hercules already distinguish the 2005-2006 auction packet revisions. | Legacy auction UI is optional work; do not conflate it with newer marketplace systems. |

These are client capability and emulator-history observations, not claims about
when every regional official server enabled the corresponding content.

## Homunculus Compatibility Notes

For `20080910`, homunculus status refreshes use the full `ZC_PROPERTY_HOMUN`
packet (`0x022E`). `ZC_HO_PAR_CHANGE` (`0x07DB`) is a later packet, starting at
`PACKETVER >= 20090610`, so it is not part of the effective 2008 rAthena map
packet table. Goro still parses the later homunculus property and param-change
variants so it can interoperate with nearby or patched servers.

Current upstream rAthena and the Goro-compatible rAthena branch make
`clif_homunculus_updatestatus` send a full `0x022E` refresh for
`PACKETVER < 20090610`. Older server checkouts without that fallback can display
stale HP, SP, or EXP values after the initial homunculus info packet.

`ZC_NOTIFY_EXP` (`0x07F6`) is the later client-side EXP gain/loss notification
packet. eAthena and Hercules both gate `clif_displayexp()` behind
`PACKETVER >= 20091027`, and Hercules' 2008 Sakray packet table does not include
`0x07F6`. Current upstream rAthena keeps the same behavior by not sending that
packet to the 2008 Sakray profile; `@showexp` remains the server text-message
path for green EXP percentage lines.

See [homunculus and mercenary support](companions-20080910.md) for the current
companion UI, AI, skill, and visual coverage.

| Opcode | Direction | Status | Scope | Goro refs |
|---:|---|---|---|---|
| `0x022E` | S->C | implemented | 2008 full homunculus property and legacy refresh path | companion_packets.go, packet.go |
| `0x07DB` | S->C | compatibility parser | 2009+ incremental homunculus param change | companion_packets.go, packet.go |
| `0x09F7` | S->C | compatibility parser | later homunculus property layout with 32-bit HP | companion_packets.go, packet.go |
| `0x0B2F` | S->C | compatibility parser | later homunculus property layout without equip item id | companion_packets.go, packet.go |
| `0x0B76` | S->C | compatibility parser | later homunculus property layout with 32-bit HP/SP | companion_packets.go, packet.go |
| `0x0BA4` | S->C | compatibility parser | later homunculus property layout with 64-bit EXP | companion_packets.go, packet.go |
| `0x0BA5` | S->C | compatibility parser | later 64-bit homunculus param change | companion_packets.go, packet.go |

## Server-Driven Display Notes

Goro handles the 2008 ShowDigit clock (`0x01B1`), Convex Mirror boss report
(`0x0293`), and NPC progress lifecycle (`0x02F0`-`0x02F2`). It also handles
the fixed Gospel/Full Strip skill notice (`0x0215`); rAthena sends that packet
to 2008 clients even though it is not declared in the effective map parser
table used to generate the full table below.

NPC and monster transformations use `ZC_NPCSPRITE_CHANGE` (`0x01B0`, 11 bytes).
Goro updates the existing actor's sprite class in both the login/bootstrap and
world packet flows, preserving movement and identity. The packet's unused type
byte is ignored, matching the original client; ordinary appearance changes
continue to use `0x00C3`/`0x01D7`.

Monk sphere updates (`0x01D0`, `0x01E1`) now update the actor's visible sphere
count. The original Super Novice public-chat prayer sends `0x01ED`; the server
checks eligibility and grants Fury. Both are included in this audit.

## Remaining player request work

| Area | Missing requests | Work implied |
|---|---|---|
| Chat-room administration | `0x00DE`, `0x00E0`, `0x00E2` | Edit room settings, transfer ownership, kick members. Receive-side room updates already exist. |
| Whisper ignore list | `0x00D3` | Request and display the server list; individual/all-player ignore toggles already work. |
| Cooking | `0x025B` | Wire the existing, currently uncalled `ParseMakingItemList` (`0x025A`) into a choice UI and send the typed cooking response. Ordinary crafting is separate and works. |
| Killer ranking | `0x0237` | Extend ranking request, response (`0x0238`) and point (`0x0236`) support where the server offers this ranking. |
| Battlegrounds | `0x02DB` | Chat plus incoming camp, score, member-position and HP packets (`0x02DC`-`0x02E0`). PvP/WoE support does not cover this UI. |
| NPC cash-point shop | `0x0288` | Old shop list, cash/Kafra balances, purchase and result UI (`0x0287` / `0x0289`). |
| Instance information UI | `0x02CF` | Incoming queue/timers/status (`0x02CB`-`0x02CE`); verify destruction control against Sakexe. Server-side instancing is a separate system. |
| Legacy auction house | `0x024B`-`0x024F`, `0x0251`, `0x025C`, `0x025D` | Eight requests, reply handling, search/listing/bidding UI, attachment and money state. Largest isolated missing protocol feature. |
| Optional protocol utilities | `0x00A2`, `0x00C1`, `0x0118` | Name lookup by character ID, server player count, explicit stop-attack. Assess whether their distinct behavior is needed; actor-name lookup and normal attack controls already exist. |

Guild alliances, opposition, relation deletion and guild chat are implemented;
the old high-priority list incorrectly still included them.

## Incoming gaps that request counts miss

These have no active handling on audited `main`; this is a prioritized list,
not an exhaustive audit of every historical server-message variant.

| Area | Packets | Current limitation |
|---|---|---|
| Combo timing | `0x01D2` | The length entry alone does not apply server-provided combo timing. |
| Devotion links | `0x01CF` | No server-owned link list applied to the scene. |
| Cast cancellation | `0x01B9` (`ZC_DISPEL`) | Despite its name, this cancels casting, not the Sage Dispel skill. No handler clears the affected actor's cast presentation when the server sends it. |
| Attack range | `0x013A` | Direct range notification is ignored; current range comes from item/skill data and attack-distance failure feedback. |
| Reward/pickup messages | `0x010A`, `0x010B`, `0x02B8` | MVP item/EXP and party pickup notifications are missing; ordinary reward state and personal pickup handling are separate. |
| Chat-room join failure | `0x00DA` | No dedicated refusal feedback. |
| Other feature replies | Cooking, auctions, cash shops, instances, battlegrounds | Need their receiving flows as well as the requests above. |

Some unsupported packets are not even in the framer's length table, so an
optional server feature can cause framing errors rather than merely hide a
window. Scope implementations around the full request/reply lifecycle.

## Login And Char Server Packets

This historical header catalog mixes client dates. It is not a 2008 denominator or a list of required features. References below were refreshed; they still do not establish an active login flow.

For rAthena `PACKETVER=20080910`, character deletion uses `0x01FB`
(`CH_DELETE_CHAR`) with a 50-byte email/key field. The older `0x0068` shape is
kept only for pre-20040419 compatibility. When testing this against local
rAthena, `conf/char_athena.conf` must use `char_del_delay: 0`; delayed deletion
is a 2010-08-03+ client flow and direct 2008 deletion otherwise fails after the
email check.

| Opcode | Header | Goro refs |
|---:|---|---|
| `0x0064` | `CA_LOGIN` | login_packets.go |
| `0x0066` | `CH_SELECT_CHAR` | login_packets.go |
| `0x0067` | `CH_MAKE_CHAR` | login_packets.go |
| `0x0068` | `CH_DELETE_CHAR` | login_packets.go |
| `0x0069` | `AC_ACCEPT_LOGIN` | login_responses.go, packet.go |
| `0x006A` | `AC_REFUSE_LOGIN` | login_responses.go, packet.go |
| `0x006B` | `HC_ACCEPT_ENTER` | login_responses.go, packet.go |
| `0x006C` | `HC_REFUSE_ENTER` | packet.go |
| `0x006D` | `HC_ACCEPT_MAKECHAR` | login_responses.go, packet.go |
| `0x006E` | `HC_REFUSE_MAKECHAR` | login_responses.go, packet.go |
| `0x006F` | `HC_ACCEPT_DELETECHAR` | login_responses.go, packet.go |
| `0x0070` | `HC_REFUSE_DELETECHAR` | login_responses.go, packet.go |
| `0x0071` | `HC_NOTIFY_ZONESVR` | login_responses.go, packet.go |
| `0x0081` | `SC_NOTIFY_BAN` | disconnect_packets.go, packet.go |
| `0x0187` | `PING` | login_packets.go, packet.go |
| `0x01DB` | `CA_REQ_HASH` | - |
| `0x01DC` | `AC_ACK_HASH` | packet.go |
| `0x01DD` | `CA_LOGIN2` | - |
| `0x01FA` | `CA_LOGIN3` | - |
| `0x01FB` | `CH_DELETE_CHAR` | login_packets.go |
| `0x0200` | `CA_CONNECT_INFO_CHANGED` | login_packets.go |
| `0x0204` | `CA_EXE_HASHCHECK` | - |
| `0x020D` | `HC_BLOCK_CHARACTER` | - |
| `0x0277` | `CA_LOGIN_PCBANG` | - |
| `0x027C` | `CA_LOGIN4` | - |
| `0x028D` | `CH_REQ_IS_VALID_CHARNAME` | - |
| `0x028E` | `HC_ACK_IS_VALID_CHARNAME` | - |
| `0x028F` | `CH_REQ_CHANGE_CHARNAME` | - |
| `0x0290` | `HC_ACK_CHANGE_CHARNAME` | - |
| `0x02B0` | `CA_LOGIN_CHANNEL` | packet.go |
| `0x0825` | `CA_SSO_LOGIN_REQ` | - |
| `0x0827` | `CH_DELETE_CHAR3_RESERVED` | - |
| `0x0828` | `HC_DELETE_CHAR3_RESERVED` | - |
| `0x0829` | `CH_DELETE_CHAR3` | - |
| `0x082A` | `HC_DELETE_CHAR3` | - |
| `0x082B` | `CH_DELETE_CHAR3_CANCEL` | - |
| `0x082C` | `HC_DELETE_CHAR3_CANCEL` | - |
| `0x082D` | `HC_ACCEPT_ENTER2` | - |
| `0x083E` | `AC_REFUSE_LOGIN` | - |
| `0x0840` | `HC_NOTIFY_ACCESSIBLE_MAPNAME` | - |
| `0x0841` | `CH_SELECT_ACCESSIBLE_MAPNAME` | - |
| `0x08B8` | `CH_SECOND_PASSWD_ACK` | - |
| `0x08B9` | `HC_SECOND_PASSWD_LOGIN` | - |
| `0x08BA` | `CH_MAKE_SECOND_PASSWD` | - |
| `0x08BE` | `CH_EDIT_SECOND_PASSWD` | - |
| `0x08C5` | `CH_AVAILABLE_SECOND_PASSWD` | - |
| `0x08D4` | `CH_REQ_CHANGE_CHARACTER_SLOT` | - |
| `0x08D5` | `HC_ACK_CHANGE_CHARACTER_SLOT` | - |
| `0x08FC` | `CH_REQ_CHANGE_CHARNAME` | - |
| `0x08FD` | `HC_ACK_CHANGE_CHARNAME` | - |
| `0x0970` | `CH_MAKE_CHAR` | - |
| `0x099D` | `HC_ACK_CHARINFO_PER_PAGE` | - |
| `0x09A0` | `HC_CHARLIST_NOTIFY` | - |
| `0x09A1` | `CH_CHARLIST_REQ` | - |
| `0x0A39` | `CH_MAKE_CHAR` | - |
| `0x0AC4` | `AC_ACCEPT_LOGIN` | - |
| `0x0AC5` | `HC_NOTIFY_ZONESVR` | - |
| `0x0B6F` | `HC_ACCEPT_MAKECHAR` | - |
| `0x0B70` | `HC_ACK_CHANGE_CHARACTER_SLOT` | - |
| `0x0B72` | `HC_ACK_CHARINFO_PER_PAGE` | - |


## Full Effective 20080910 Map Packet DB

| Opcode | Registration | Status | Symbol | Length | rAthena handler | Goro refs |
|---:|---|---|---|---:|---|---|
| `0x0064` | length | referenced | `0x0064` | `55` | `-` | login_packets.go |
| `0x0065` | length | referenced | `0x0065` | `17` | `-` | login_packets.go |
| `0x0066` | length | referenced | `0x0066` | `3` | `-` | login_packets.go |
| `0x0067` | length | referenced | `0x0067` | `37` | `-` | login_packets.go |
| `0x0068` | length | referenced | `0x0068` | `46` | `-` | login_packets.go |
| `0x0069` | length | referenced | `0x0069` | `-1` | `-` | login_responses.go, packet.go |
| `0x006A` | length | referenced | `0x006a` | `23` | `-` | login_responses.go, packet.go |
| `0x006B` | length | referenced | `0x006b` | `-1` | `-` | login_responses.go, packet.go |
| `0x006C` | length | referenced | `0x006c` | `3` | `-` | packet.go |
| `0x006D` | length | referenced | `0x006d` | `110` | `-` | login_responses.go, packet.go |
| `0x006E` | length | referenced | `0x006e` | `3` | `-` | login_responses.go, packet.go |
| `0x006F` | length | referenced | `0x006f` | `2` | `-` | login_responses.go, packet.go |
| `0x0070` | length | referenced | `0x0070` | `3` | `-` | login_responses.go, packet.go |
| `0x0071` | length | referenced | `0x0071` | `28` | `-` | login_responses.go, packet.go |
| `0x0072` | request | alias-covered | `0x0072` | `25` | `clif_parse_UseSkillToId` | - |
| `0x0075` | length | untracked | `0x0075` | `-1` | `-` | - |
| `0x0076` | length | untracked | `0x0076` | `9` | `-` | - |
| `0x0077` | length | referenced | `0x0077` | `5` | `-` | packet.go |
| `0x0079` | length | referenced | `0x0079` | `53` | `-` | actor_packets.go, packet.go |
| `0x007A` | length | referenced | `0x007a` | `58` | `-` | actor_packets.go, packet.go |
| `0x007B` | length | referenced | `0x007b` | `60` | `-` | actor_packets.go, packet.go |
| `0x007C` | length | referenced | `0x007c` | `42` | `-` | actor_packets.go, packet.go |
| `0x007D` | request | sender | `0x007d` | `2` | `clif_parse_LoadEndAck` | login_packets.go |
| `0x007E` | request | sender | `0x007e` | `102` | `clif_parse_UseSkillToPosMoreInfo` | packet.go, skill_packets.go |
| `0x0082` | length | untracked | `0x0082` | `2` | `-` | - |
| `0x0083` | length | untracked | `0x0083` | `2` | `-` | - |
| `0x0084` | length | untracked | `0x0084` | `2` | `-` | - |
| `0x0085` | request | sender | `0x0085` | `11` | `clif_parse_ChangeDir` | login_packets.go |
| `0x0089` | request | sender | `0x0089` | `8` | `clif_parse_TickSend` | login_packets.go |
| `0x008B` | length | referenced | `0x008b` | `2` | `-` | packet.go |
| `0x008C` | request | sender | `0x008c` | `11` | `clif_parse_GetCharNameRequest` | chat_packets.go, login_packets.go |
| `0x008D` | length | referenced | `0x008d` | `-1` | `-` | chat_packets.go, packet.go |
| `0x008E` | length | referenced | `0x008e` | `-1` | `-` | chat_packets.go, packet.go |
| `0x0090` | request | sender | `HEADER_CZ_CONTACTNPC` | `sizeof( PACKET_CZ_CONTACTNPC )` | `clif_parse_NpcClicked` | npc_packets.go |
| `0x0093` | length | untracked | `0x0093` | `2` | `-` | - |
| `0x0094` | request | sender | `0x0094` | `14` | `clif_parse_MoveToKafra` | item_packets.go, login_packets.go |
| `0x0096` | request | sender | `0x0096` | `-1` | `clif_parse_WisMessage` | chat_packets.go |
| `0x0099` | request | admin gap | `HEADER_CZ_BROADCAST` | `-1` | `clif_parse_Broadcast` | packet.go |
| `0x009B` | request | alias-covered | `0x009b` | `26` | `clif_parse_WantToConnection` | login_packets.go, packet.go |
| `0x009E` | length | referenced | `0x009e` | `17` | `-` | item_packets.go, packet.go |
| `0x009F` | request | alias-covered | `0x009f` | `14` | `clif_parse_UseItem` | item_packets.go |
| `0x00A2` | request | missing | `0x00a2` | `15` | `clif_parse_SolveCharName` | item_packets.go |
| `0x00A7` | request | sender | `0x00a7` | `8` | `clif_parse_WalkToXY` | item_packets.go, login_packets.go, packet.go |
| `0x00A9` | request | sender | `HEADER_CZ_REQ_WEAR_EQUIP` | `sizeof( PACKET_CZ_REQ_WEAR_EQUIP )` | `clif_parse_EquipItem` | item_packets.go, packet.go |
| `0x00AB` | request | sender | `0x00ab` | `4` | `clif_parse_UnequipItem` | item_packets.go |
| `0x00AE` | length | untracked | `0x00ae` | `-1` | `-` | - |
| `0x00B2` | request | sender | `0x00b2` | `3` | `clif_parse_Restart` | login_packets.go, packet.go |
| `0x00B8` | request | sender | `0x00b8` | `7` | `clif_parse_NpcSelectMenu` | npc_packets.go |
| `0x00B9` | request | sender | `0x00b9` | `6` | `clif_parse_NpcNextClicked` | npc_packets.go |
| `0x00BA` | length | referenced | `0x00ba` | `2` | `-` | packet.go |
| `0x00BB` | request | sender | `0x00bb` | `5` | `clif_parse_StatusUp` | packet.go, status_packets.go |
| `0x00BF` | request | sender | `HEADER_CZ_REQ_EMOTION` | `sizeof( PACKET_CZ_REQ_EMOTION )` | `clif_parse_Emotion` | emotion_packets.go, packet.go |
| `0x00C1` | request | missing | `0x00c1` | `2` | `clif_parse_HowManyConnections` | packet.go |
| `0x00C3` | length | referenced | `0x00c3` | `8` | `-` | actor_packets.go, packet.go |
| `0x00C5` | request | sender | `HEADER_CZ_ACK_SELECT_DEALTYPE` | `sizeof( PACKET_CZ_ACK_SELECT_DEALTYPE )` | `clif_parse_NpcBuySellSelected` | item_packets.go, packet.go |
| `0x00C6` | length | referenced | `0x00c6` | `-1` | `-` | item_packets.go, packet.go |
| `0x00C8` | request | sender | `0x00c8` | `-1` | `clif_parse_NpcBuyListSend` | item_packets.go, packet.go |
| `0x00C9` | request | sender | `HEADER_CZ_PC_SELL_ITEMLIST` | `-1` | `clif_parse_NpcSellListSend` | item_packets.go, packet.go |
| `0x00CA` | length | referenced | `0x00ca` | `3` | `-` | item_packets.go, packet.go |
| `0x00CB` | length | referenced | `0x00cb` | `3` | `-` | item_packets.go, packet.go |
| `0x00CC` | request | admin gap | `0x00cc` | `6` | `clif_parse_GMKick` | - |
| `0x00CD` | length | untracked | `0x00cd` | `3` | `-` | - |
| `0x00CE` | request | admin gap | `0x00ce` | `2` | `clif_parse_GMKickAll` | - |
| `0x00CF` | request | sender | `0x00cf` | `27` | `clif_parse_PMIgnore` | chat_packets.go, packet.go |
| `0x00D0` | request | sender | `HEADER_CZ_SETTING_WHISPER_STATE` | `sizeof( PACKET_CZ_SETTING_WHISPER_STATE )` | `clif_parse_PMIgnoreAll` | chat_packets.go, packet.go |
| `0x00D3` | request | missing | `0x00d3` | `2` | `clif_parse_PMIgnoreList` | packet.go |
| `0x00D5` | request | sender | `HEADER_CZ_CREATE_CHATROOM` | `-1` | `clif_parse_CreateChatRoom` | chat_packets.go, packet.go |
| `0x00D9` | request | sender | `HEADER_CZ_REQ_ENTER_ROOM` | `sizeof( PACKET_CZ_REQ_ENTER_ROOM )` | `clif_parse_ChatAddMember` | chat_packets.go, packet.go |
| `0x00DE` | request | missing | `HEADER_CZ_CHANGE_CHATROOM` | `-1` | `clif_parse_ChatRoomStatusChange` | packet.go |
| `0x00E0` | request | missing | `0x00e0` | `30` | `clif_parse_ChangeChatOwner` | packet.go |
| `0x00E2` | request | missing | `0x00e2` | `26` | `clif_parse_KickFromChat` | packet.go |
| `0x00E3` | request | sender | `0x00e3` | `2` | `clif_parse_ChatLeave` | chat_packets.go, packet.go |
| `0x00E4` | request | sender | `0x00e4` | `6` | `clif_parse_TradeRequest` | packet.go, trade_packets.go |
| `0x00E5` | length | referenced | `0x00e5` | `26` | `-` | packet.go, trade_packets.go |
| `0x00E6` | request | sender | `0x00e6` | `3` | `clif_parse_TradeAck` | packet.go, trade_packets.go |
| `0x00E8` | request | sender | `HEADER_CZ_ADD_EXCHANGE_ITEM` | `sizeof( PACKET_CZ_ADD_EXCHANGE_ITEM )` | `clif_parse_TradeAddItem` | packet.go, trade_packets.go |
| `0x00EA` | length | referenced | `0x00ea` | `5` | `-` | packet.go, trade_packets.go |
| `0x00EB` | request | sender | `0x00eb` | `2` | `clif_parse_TradeOk` | packet.go, trade_packets.go |
| `0x00ED` | request | sender | `0x00ed` | `2` | `clif_parse_TradeCancel` | packet.go, trade_packets.go |
| `0x00EF` | request | sender | `0x00ef` | `2` | `clif_parse_TradeCommit` | packet.go, trade_packets.go |
| `0x00F3` | request | sender | `0x00f3` | `-1` | `clif_parse_GlobalMessage` | chat_packets.go, item_packets.go, login_packets.go, packet.go |
| `0x00F5` | request | sender | `0x00f5` | `8` | `clif_parse_TakeItem` | item_packets.go, packet.go |
| `0x00F7` | request | sender | `0x00f7` | `22` | `clif_parse_MoveFromKafra` | item_packets.go, packet.go |
| `0x00F9` | request | sender | `HEADER_CZ_MAKE_GROUP` | `sizeof( PACKET_CZ_MAKE_GROUP )` | `clif_parse_CreateParty` | party_packets.go |
| `0x00FB` | length | referenced | `0x00fb` | `-1` | `-` | packet.go, party_packets.go |
| `0x00FC` | request | sender | `HEADER_CZ_REQ_JOIN_GROUP` | `sizeof( PACKET_CZ_REQ_JOIN_GROUP )` | `clif_parse_PartyInvite` | actor_packets.go, party_packets.go |
| `0x00FD` | length | referenced | `0x00fd` | `27` | `-` | packet.go, party_packets.go |
| `0x00FF` | request | alias-covered | `HEADER_CZ_JOIN_GROUP` | `sizeof( PACKET_CZ_JOIN_GROUP )` | `clif_parse_ReplyPartyInvite` | chat_packets.go, login_packets.go, party_packets.go |
| `0x0100` | request | sender | `HEADER_CZ_REQ_LEAVE_GROUP` | `sizeof( PACKET_CZ_REQ_LEAVE_GROUP )` | `clif_parse_LeaveParty` | party_packets.go |
| `0x0101` | length | referenced | `0x0101` | `6` | `-` | packet.go, party_packets.go |
| `0x0102` | request | sender | `0x0102` | `6` | `clif_parse_PartyChangeOption` | party_packets.go |
| `0x0103` | request | sender | `HEADER_CZ_REQ_EXPEL_GROUP_MEMBER` | `sizeof( PACKET_CZ_REQ_EXPEL_GROUP_MEMBER )` | `clif_parse_RemovePartyMember` | party_packets.go |
| `0x0104` | length | referenced | `0x0104` | `79` | `-` | packet.go, party_packets.go |
| `0x0108` | request | sender | `0x0108` | `-1` | `clif_parse_PartyMessage` | party_packets.go |
| `0x0109` | length | referenced | `0x0109` | `-1` | `-` | packet.go, party_packets.go |
| `0x0112` | request | sender | `0x0112` | `4` | `clif_parse_SkillUp` | skill_packets.go |
| `0x0113` | request | sender | `0x0113` | `22` | `clif_parse_UseSkillToPos` | item_packets.go, packet.go, skill_packets.go |
| `0x0114` | length | referenced | `0x0114` | `31` | `-` | actor_packets.go, packet.go |
| `0x0115` | length | referenced | `0x0115` | `35` | `-` | packet.go |
| `0x0116` | request | sender | `0x0116` | `10` | `clif_parse_DropItem` | item_packets.go, packet.go, skill_packets.go |
| `0x0118` | request | missing | `0x0118` | `2` | `clif_parse_StopAttack` | packet.go |
| `0x0119` | length | referenced | `0x0119` | `13` | `-` | actor_packets.go, packet.go |
| `0x011B` | request | sender | `HEADER_CZ_SELECT_WARPPOINT` | `sizeof( PACKET_CZ_SELECT_WARPPOINT )` | `clif_parse_UseSkillMap` | skill_packets.go |
| `0x011D` | request | sender | `0x011d` | `2` | `clif_parse_RequestMemo` | packet.go, skill_packets.go |
| `0x011F` | length | referenced | `0x011f` | `16` | `-` | packet.go, skill_packets.go |
| `0x0126` | request | sender | `HEADER_CZ_MOVE_ITEM_FROM_BODY_TO_CART` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_BODY_TO_CART )` | `clif_parse_PutItemToCart` | item_packets.go |
| `0x0127` | request | sender | `HEADER_CZ_MOVE_ITEM_FROM_CART_TO_BODY` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_CART_TO_BODY )` | `clif_parse_GetItemFromCart` | item_packets.go |
| `0x0128` | request | sender | `HEADER_CZ_MOVE_ITEM_FROM_STORE_TO_CART` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_STORE_TO_CART )` | `clif_parse_MoveFromKafraToCart` | item_packets.go |
| `0x0129` | request | sender | `HEADER_CZ_MOVE_ITEM_FROM_CART_TO_STORE` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_CART_TO_STORE )` | `clif_parse_MoveToKafraFromCart` | item_packets.go |
| `0x012A` | request | sender | `0x012a` | `2` | `clif_parse_RemoveOption` | equipment_packets.go |
| `0x012E` | request | sender | `0x012e` | `2` | `clif_parse_CloseVending` | vending_packets.go |
| `0x012F` | request | alias-covered | `0x012f` | `-1` | `clif_parse_OpenVending` | vending_packets.go |
| `0x0130` | request | sender | `0x0130` | `6` | `clif_parse_VendingListReq` | vending_packets.go |
| `0x0134` | request | sender | `HEADER_CZ_PC_PURCHASE_ITEMLIST_FROMMC` | `-1` | `clif_parse_PurchaseReq` | vending_packets.go |
| `0x0138` | length | untracked | `0x0138` | `3` | `-` | - |
| `0x013F` | request | admin gap | `0x013f` | `26` | `clif_parse_GM_Item_Monster` | - |
| `0x0140` | request | admin gap | `HEADER_CZ_MOVETO_MAP` | `sizeof( PACKET_CZ_MOVETO_MAP )` | `clif_parse_MapMove` | - |
| `0x0143` | request | sender | `HEADER_CZ_INPUT_EDITDLG` | `sizeof( PACKET_CZ_INPUT_EDITDLG )` | `clif_parse_NpcAmountInput` | npc_packets.go |
| `0x0145` | length | referenced | `0x0145` | `19` | `-` | packet.go |
| `0x0146` | request | sender | `HEADER_CZ_CLOSE_DIALOG` | `sizeof( PACKET_CZ_CLOSE_DIALOG )` | `clif_parse_NpcCloseClicked` | npc_packets.go |
| `0x0147` | length | referenced | `0x0147` | `39` | `-` | packet.go, skill_packets.go |
| `0x0149` | request | admin gap | `0x0149` | `9` | `clif_parse_GMReqNoChat` | - |
| `0x014A` | length | referenced | `0x014a` | `6` | `-` | packet.go |
| `0x014D` | request | sender | `0x014d` | `2` | `clif_parse_GuildCheckMaster` | guild_packets.go |
| `0x014F` | request | sender | `0x014f` | `6` | `clif_parse_GuildRequestInfo` | guild_packets.go |
| `0x0150` | length | referenced | `0x0150` | `110` | `-` | guild_packets.go, packet.go |
| `0x0151` | request | sender | `0x0151` | `6` | `clif_parse_GuildRequestEmblem` | guild_packets.go |
| `0x0152` | length | referenced | `0x0152` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0153` | request | sender | `0x0153` | `-1` | `clif_parse_GuildChangeEmblem` | guild_packets.go |
| `0x0154` | length | referenced | `0x0154` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0155` | request | sender | `HEADER_CZ_REQ_CHANGE_MEMBERPOS` | `-1` | `clif_parse_GuildChangeMemberPosition` | guild_packets.go |
| `0x0157` | length | referenced | `0x0157` | `6` | `-` | guild_packets.go |
| `0x0158` | length | referenced | `0x0158` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0159` | request | sender | `HEADER_CZ_REQ_LEAVE_GUILD` | `sizeof( PACKET_CZ_REQ_LEAVE_GUILD )` | `clif_parse_GuildLeave` | guild_packets.go |
| `0x015B` | request | sender | `HEADER_CZ_REQ_BAN_GUILD` | `sizeof( PACKET_CZ_REQ_BAN_GUILD )` | `clif_parse_GuildExpulsion` | guild_packets.go |
| `0x015D` | request | sender | `HEADER_CZ_REQ_DISORGANIZE_GUILD` | `sizeof( PACKET_CZ_REQ_DISORGANIZE_GUILD )` | `clif_parse_GuildBreak` | guild_packets.go |
| `0x015F` | length | referenced | `0x015f` | `42` | `-` | packet.go |
| `0x0161` | request | sender | `0x0161` | `-1` | `clif_parse_GuildChangePositionInfo` | guild_packets.go |
| `0x0164` | length | untracked | `0x0164` | `-1` | `-` | - |
| `0x0165` | request | sender | `0x0165` | `30` | `clif_parse_CreateGuild` | guild_packets.go |
| `0x0166` | length | referenced | `0x0166` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0168` | request | sender | `HEADER_CZ_REQ_JOIN_GUILD` | `sizeof( PACKET_CZ_REQ_JOIN_GUILD )` | `clif_parse_GuildInvite` | guild_packets.go |
| `0x016B` | request | sender | `HEADER_CZ_JOIN_GUILD` | `sizeof( PACKET_CZ_JOIN_GUILD )` | `clif_parse_GuildReplyInvite` | guild_packets.go |
| `0x016C` | length | referenced | `0x016c` | `43` | `-` | guild_packets.go, packet.go |
| `0x016E` | request | sender | `0x016e` | `186` | `clif_parse_GuildChangeNotice` | guild_packets.go |
| `0x0170` | request | sender | `0x0170` | `14` | `clif_parse_GuildRequestAlliance` | guild_packets.go |
| `0x0172` | request | sender | `0x0172` | `10` | `clif_parse_GuildReplyAlliance` | guild_packets.go |
| `0x0175` | length | referenced | `0x0175` | `6` | `-` | guild_packets.go |
| `0x0176` | length | referenced | `0x0176` | `106` | `-` | guild_packets.go, packet.go |
| `0x0177` | length | referenced | `0x0177` | `-1` | `-` | item_packets.go, packet.go |
| `0x0178` | request | sender | `HEADER_CZ_REQ_ITEMIDENTIFY` | `sizeof( PACKET_CZ_REQ_ITEMIDENTIFY )` | `clif_parse_ItemIdentify` | item_packets.go |
| `0x017A` | request | sender | `HEADER_CZ_REQ_ITEMCOMPOSITION_LIST` | `sizeof( PACKET_CZ_REQ_ITEMCOMPOSITION_LIST )` | `clif_parse_UseCard` | item_packets.go |
| `0x017B` | length | referenced | `0x017b` | `-1` | `-` | item_packets.go, packet.go |
| `0x017C` | request | sender | `HEADER_CZ_REQ_ITEMCOMPOSITION` | `sizeof( PACKET_CZ_REQ_ITEMCOMPOSITION )` | `clif_parse_InsertCard` | item_packets.go |
| `0x017E` | request | sender | `0x017e` | `-1` | `clif_parse_GuildMessage` | guild_packets.go |
| `0x0180` | request | sender | `0x0180` | `6` | `clif_parse_GuildOpposition` | guild_packets.go |
| `0x0182` | length | referenced | `0x0182` | `106` | `-` | guild_packets.go, packet.go |
| `0x0183` | request | sender | `0x0183` | `10` | `clif_parse_GuildDelAlliance` | guild_packets.go |
| `0x0185` | length | referenced | `0x0185` | `34` | `-` | guild_packets.go, packet.go |
| `0x0187` | length | referenced | `0x0187` | `6` | `-` | login_packets.go, packet.go |
| `0x018A` | request | sender | `0x018a` | `4` | `clif_parse_QuitGame` | login_packets.go |
| `0x018B` | length | referenced | `0x018b` | `4` | `-` | packet.go, restart_packets.go |
| `0x018E` | request | sender | `HEADER_CZ_REQMAKINGITEM` | `sizeof( struct PACKET_CZ_REQMAKINGITEM )` | `clif_parse_ProduceMix` | item_packets.go |
| `0x0190` | request | sender | `0x0190` | `19` | `clif_parse_ActionRequest` | item_packets.go, login_packets.go, skill_packets.go |
| `0x0193` | request | sender | `0x0193` | `2` | `clif_parse_CloseKafra` | item_packets.go |
| `0x0196` | length | referenced | `0x0196` | `9` | `-` | packet.go, status_packets.go |
| `0x0197` | request | admin gap | `HEADER_CZ_RESET` | `sizeof( PACKET_CZ_RESET )` | `clif_parse_ResetChar` | packet.go |
| `0x0198` | request | admin gap | `0x0198` | `8` | `clif_parse_GMChangeMapType` | - |
| `0x0199` | length | referenced | `0x0199` | `4` | `-` | packet.go, pvp_packets.go |
| `0x019A` | length | referenced | `0x019a` | `14` | `-` | packet.go, pvp_packets.go |
| `0x019C` | request | admin gap | `HEADER_CZ_LOCALBROADCAST` | `-1` | `clif_parse_LocalBroadcast` | packet.go |
| `0x019D` | request | admin gap | `0x019d` | `6` | `clif_parse_GMHide` | packet.go |
| `0x019F` | request | sender | `0x019f` | `6` | `clif_parse_CatchPet` | pet_packets.go |
| `0x01A1` | request | sender | `0x01a1` | `3` | `clif_parse_PetMenu` | pet_packets.go |
| `0x01A3` | length | referenced | `0x01a3` | `5` | `-` | packet.go, pet_packets.go |
| `0x01A5` | request | sender | `0x01a5` | `26` | `clif_parse_ChangePetName` | pet_packets.go |
| `0x01A6` | length | referenced | `0x01a6` | `-1` | `-` | packet.go, pet_packets.go |
| `0x01A7` | request | sender | `0x01a7` | `4` | `clif_parse_SelectEgg` | pet_packets.go |
| `0x01A8` | length | referenced | `0x01a8` | `4` | `-` | packet.go |
| `0x01A9` | request | sender | `0x01a9` | `6` | `clif_parse_SendEmotion` | pet_packets.go |
| `0x01AC` | length | referenced | `0x01ac` | `6` | `-` | skill_packets.go |
| `0x01AD` | length | referenced | `0x01ad` | `-1` | `-` | item_packets.go, packet.go |
| `0x01AE` | request | sender | `HEADER_CZ_REQ_MAKINGARROW` | `sizeof( PACKET_CZ_REQ_MAKINGARROW )` | `clif_parse_SelectArrow` | item_packets.go |
| `0x01AF` | request | sender | `0x01af` | `4` | `clif_parse_ChangeCart` | skill_packets.go |
| `0x01B0` | length | referenced | `0x01b0` | `11` | `-` | actor_packets.go, packet.go |
| `0x01B1` | length | referenced | `0x01b1` | `7` | `-` | packet.go, server_info_packets.go |
| `0x01B2` | request | sender | `0x01b2` | `-1` | `clif_parse_OpenVending` | vending_packets.go |
| `0x01B5` | length | referenced | `0x01b5` | `18` | `-` | packet.go |
| `0x01B6` | length | referenced | `0x01b6` | `114` | `-` | guild_packets.go, packet.go |
| `0x01B7` | length | untracked | `0x01b7` | `6` | `-` | - |
| `0x01B8` | length | untracked | `0x01b8` | `3` | `-` | - |
| `0x01BA` | request | admin gap | `0x01ba` | `26` | `clif_parse_GMShift` | - |
| `0x01BB` | request | admin gap | `0x01bb` | `26` | `clif_parse_GMShift` | - |
| `0x01BC` | request | admin gap | `0x01bc` | `26` | `clif_parse_GMRecall` | - |
| `0x01BD` | request | admin gap | `0x01bd` | `26` | `clif_parse_GMRecall` | - |
| `0x01BE` | length | untracked | `0x01be` | `2` | `-` | - |
| `0x01BF` | length | untracked | `0x01bf` | `3` | `-` | - |
| `0x01C0` | length | untracked | `0x01c0` | `2` | `-` | - |
| `0x01C1` | length | untracked | `0x01c1` | `14` | `-` | - |
| `0x01C2` | length | untracked | `0x01c2` | `10` | `-` | - |
| `0x01C3` | length | referenced | `0x01c3` | `-1` | `-` | chat_packets.go, packet.go |
| `0x01C6` | length | untracked | `0x01c6` | `4` | `-` | - |
| `0x01C7` | length | untracked | `0x01c7` | `2` | `-` | - |
| `0x01C8` | length | referenced | `useItemAckType` | `sizeof( struct PACKET_ZC_USE_ITEM_ACK )` | `-` | item_packets.go, packet.go |
| `0x01CA` | length | referenced | `0x01ca` | `3` | `-` | packet.go |
| `0x01CB` | length | untracked | `0x01cb` | `9` | `-` | - |
| `0x01CC` | length | untracked | `0x01cc` | `9` | `-` | - |
| `0x01CE` | request | sender | `HEADER_CZ_SELECTAUTOSPELL` | `sizeof( PACKET_CZ_SELECTAUTOSPELL )` | `clif_parse_AutoSpell` | skill_packets.go |
| `0x01CF` | length | referenced | `0x01cf` | `28` | `-` | packet.go |
| `0x01D0` | length | referenced | `0x01d0` | `8` | `-` | effect_packets.go, packet.go |
| `0x01D5` | request | sender | `HEADER_CZ_INPUT_EDITDLGSTR` | `-1` | `clif_parse_NpcStringInput` | npc_packets.go |
| `0x01D7` | length | referenced | `0x01d7` | `11` | `-` | actor_packets.go, packet.go |
| `0x01D8` | length | referenced | `0x01d8` | `54` | `-` | actor_packets.go, packet.go |
| `0x01D9` | length | referenced | `0x01d9` | `53` | `-` | actor_packets.go, packet.go |
| `0x01DA` | length | referenced | `0x01da` | `60` | `-` | actor_packets.go, packet.go |
| `0x01DB` | length | untracked | `0x01db` | `2` | `-` | - |
| `0x01DC` | length | referenced | `0x01dc` | `-1` | `-` | packet.go |
| `0x01DD` | length | untracked | `0x01dd` | `47` | `-` | - |
| `0x01DE` | length | referenced | `0x01de` | `33` | `-` | actor_packets.go, packet.go |
| `0x01DF` | request | admin gap | `0x01df` | `6` | `clif_parse_GMReqAccountName` | - |
| `0x01E0` | length | untracked | `0x01e0` | `30` | `-` | - |
| `0x01E1` | length | referenced | `0x01e1` | `8` | `-` | effect_packets.go, packet.go |
| `0x01E2` | length | untracked | `0x01e2` | `34` | `-` | - |
| `0x01E3` | length | untracked | `0x01e3` | `14` | `-` | - |
| `0x01E4` | length | untracked | `0x01e4` | `2` | `-` | - |
| `0x01E5` | length | untracked | `0x01e5` | `6` | `-` | - |
| `0x01E6` | length | referenced | `0x01e6` | `26` | `-` | packet.go |
| `0x01E7` | request | sender | `0x01e7` | `2` | `clif_parse_NoviceDoriDori` | novice_packets.go |
| `0x01E8` | request | sender | `HEADER_CZ_MAKE_GROUP2` | `sizeof( PACKET_CZ_MAKE_GROUP2 )` | `clif_parse_CreateParty2` | party_packets.go |
| `0x01EC` | length | untracked | `0x01ec` | `26` | `-` | - |
| `0x01ED` | request | sender | `0x01ed` | `2` | `clif_parse_NoviceExplosionSpirits` | novice_packets.go |
| `0x01F0` | length | untracked | `0x01f0` | `-1` | `-` | - |
| `0x01F1` | length | referenced | `0x01f1` | `-1` | `-` | packet.go |
| `0x01F3` | length | referenced | `0x01f3` | `10` | `-` | effect_packets.go, packet.go |
| `0x01F6` | length | referenced | `0x01f6` | `34` | `-` | adoption_packets.go, packet.go |
| `0x01F7` | request | sender | `HEADER_CZ_JOIN_BABY` | `sizeof( PACKET_CZ_JOIN_BABY )` | `clif_parse_Adopt_reply` | adoption_packets.go |
| `0x01F8` | length | referenced | `0x01f8` | `2` | `-` | adoption_packets.go, packet.go |
| `0x01F9` | request | sender | `0x01f9` | `6` | `clif_parse_Adopt_request` | adoption_packets.go |
| `0x01FA` | length | untracked | `0x01fa` | `48` | `-` | - |
| `0x01FB` | length | referenced | `0x01fb` | `56` | `-` | login_packets.go |
| `0x01FC` | length | referenced | `0x01fc` | `-1` | `-` | item_packets.go, packet.go |
| `0x01FD` | request | sender | `HEADER_CZ_REQ_ITEMREPAIR1` | `sizeof( struct PACKET_CZ_REQ_ITEMREPAIR1 )` | `clif_parse_RepairItem` | item_packets.go |
| `0x0200` | length | referenced | `0x0200` | `26` | `-` | login_packets.go |
| `0x0201` | length | referenced | `0x0201` | `-1` | `-` | friend_packets.go, packet.go |
| `0x0202` | request | sender | `0x0202` | `26` | `clif_parse_FriendsListAdd` | friend_packets.go |
| `0x0203` | request | sender | `0x0203` | `10` | `clif_parse_FriendsListRemove` | friend_packets.go |
| `0x0204` | length | untracked | `0x0204` | `18` | `-` | - |
| `0x0207` | length | referenced | `0x0207` | `34` | `-` | friend_packets.go, packet.go |
| `0x0208` | request | sender | `0x0208` | `14` | `clif_parse_FriendsListReply` | friend_packets.go |
| `0x0209` | length | referenced | `0x0209` | `36` | `-` | friend_packets.go, packet.go |
| `0x020A` | length | referenced | `0x020a` | `10` | `-` | friend_packets.go, packet.go |
| `0x020D` | length | untracked | `0x020d` | `-1` | `-` | - |
| `0x020E` | length | referenced | `0x020e` | `32` | `-` | packet.go, taekwon_packets.go |
| `0x020F` | request | sender | `0x020f` | `10` | `clif_parse_PVPInfo` | packet.go, pvp_packets.go |
| `0x0210` | length | referenced | `0x0210` | `22` | `-` | packet.go, pvp_packets.go |
| `0x0212` | request | admin gap | `0x0212` | `26` | `clif_parse_GMRc` | - |
| `0x0213` | request | admin gap | `0x0213` | `26` | `clif_parse_Check` | - |
| `0x0214` | length | referenced | `0x0214` | `42` | `-` | packet.go |
| `0x0216` | length | referenced | `0x0216` | `6` | `-` | adoption_packets.go, packet.go |
| `0x0217` | request | sender | `HEADER_CZ_BLACKSMITH_RANK` | `sizeof( PACKET_CZ_BLACKSMITH_RANK )` | `clif_parse_ranklist_blacksmith` | ranking_packets.go |
| `0x0218` | request | sender | `HEADER_CZ_ALCHEMIST_RANK` | `sizeof( PACKET_CZ_ALCHEMIST_RANK )` | `clif_parse_ranklist_alchemist` | ranking_packets.go |
| `0x021D` | request | sender | `HEADER_CZ_LESSEFFECT` | `sizeof( PACKET_CZ_LESSEFFECT )` | `clif_parse_LessEffect` | effect_packets.go |
| `0x021E` | length | referenced | `0x021e` | `6` | `-` | effect_packets.go, packet.go |
| `0x021F` | length | untracked | `0x021f` | `66` | `-` | - |
| `0x0220` | length | referenced | `0x0220` | `10` | `-` | packet.go |
| `0x0221` | length | referenced | `0x0221` | `-1` | `-` | item_packets.go, packet.go |
| `0x0222` | request | sender | `0x0222` | `6` | `clif_parse_WeaponRefine` | item_packets.go |
| `0x0225` | request | sender | `HEADER_CZ_TAEKWON_RANK` | `sizeof( PACKET_CZ_TAEKWON_RANK )` | `clif_parse_ranklist_taekwon` | ranking_packets.go |
| `0x0227` | length | untracked | `0x0227` | `18` | `-` | - |
| `0x0228` | length | untracked | `0x0228` | `18` | `-` | - |
| `0x0229` | length | referenced | `0x0229` | `15` | `-` | actor_packets.go, packet.go |
| `0x022A` | length | referenced | `0x022a` | `58` | `-` | packet.go |
| `0x022B` | length | referenced | `0x022b` | `57` | `-` | packet.go |
| `0x022C` | length | referenced | `0x022c` | `65` | `-` | actor_packets.go, packet.go |
| `0x022D` | request | sender | `0x022d` | `5` | `clif_parse_HomMenu` | companion_packets.go |
| `0x022E` | length | referenced | `0x022e` | `71` | `-` | companion_packets.go, packet.go |
| `0x022F` | length | referenced | `0x022f` | `5` | `-` | client.go, companion_packets.go, packet.go |
| `0x0231` | request | sender | `0x0231` | `26` | `clif_parse_ChangeHomunculusName` | companion_packets.go |
| `0x0232` | request | sender | `HEADER_CZ_REQUEST_MOVENPC` | `sizeof( PACKET_CZ_REQUEST_MOVENPC )` | `clif_parse_HomMoveTo` | companion_packets.go |
| `0x0233` | request | sender | `0x0233` | `11` | `clif_parse_HomAttack` | companion_packets.go |
| `0x0234` | request | sender | `0x0234` | `6` | `clif_parse_HomMoveToMaster` | companion_packets.go |
| `0x0237` | request | missing | `HEADER_CZ_KILLER_RANK` | `sizeof( PACKET_CZ_KILLER_RANK )` | `clif_parse_ranklist_killer` | - |
| `0x023A` | length | referenced | `0x023a` | `4` | `-` | packet.go, storage_password_packets.go |
| `0x023B` | request | sender | `0x023b` | `36` | `clif_parse_StoragePassword` | storage_password_packets.go |
| `0x023C` | length | referenced | `0x023c` | `6` | `-` | packet.go, storage_password_packets.go |
| `0x023D` | length | untracked | `0x023d` | `-1` | `-` | - |
| `0x023E` | length | untracked | `0x023e` | `8` | `-` | - |
| `0x023F` | request | sender | `0x023f` | `2` | `clif_parse_Mail_refreshinbox` | mail_packets.go |
| `0x0240` | length | referenced | `0x0240` | `-1` | `-` | mail_packets.go, packet.go |
| `0x0241` | request | sender | `0x0241` | `6` | `clif_parse_Mail_read` | mail_packets.go |
| `0x0242` | length | referenced | `0x0242` | `-1` | `-` | mail_packets.go, packet.go |
| `0x0243` | request | sender | `0x0243` | `6` | `clif_parse_Mail_delete` | mail_packets.go |
| `0x0244` | request | sender | `0x0244` | `6` | `clif_parse_Mail_getattach` | mail_packets.go |
| `0x0245` | length | referenced | `0x0245` | `3` | `-` | mail_packets.go, packet.go |
| `0x0246` | request | sender | `0x0246` | `4` | `clif_parse_Mail_winopen` | mail_packets.go |
| `0x0247` | request | sender | `0x0247` | `8` | `clif_parse_Mail_setattach` | mail_packets.go |
| `0x0248` | request | sender | `0x0248` | `-1` | `clif_parse_Mail_send` | mail_packets.go |
| `0x0249` | length | referenced | `0x0249` | `3` | `-` | mail_packets.go, packet.go |
| `0x024A` | length | referenced | `0x024a` | `70` | `-` | mail_packets.go, packet.go |
| `0x024B` | request | missing | `0x024b` | `4` | `clif_parse_Auction_cancelreg` | - |
| `0x024C` | request | missing | `0x024c` | `8` | `clif_parse_Auction_setitem` | - |
| `0x024D` | request | missing | `HEADER_CZ_AUCTION_ADD` | `sizeof( PACKET_CZ_AUCTION_ADD )` | `clif_parse_Auction_register` | - |
| `0x024E` | request | missing | `0x024e` | `6` | `clif_parse_Auction_cancel` | - |
| `0x024F` | request | missing | `HEADER_CZ_AUCTION_BUY` | `sizeof( PACKET_CZ_AUCTION_BUY )` | `clif_parse_Auction_bid` | - |
| `0x0250` | length | untracked | `0x0250` | `3` | `-` | - |
| `0x0251` | request | missing | `HEADER_CZ_AUCTION_ITEM_SEARCH` | `sizeof( PACKET_CZ_AUCTION_ITEM_SEARCH )` | `clif_parse_Auction_search` | - |
| `0x0252` | length | untracked | `0x0252` | `-1` | `-` | - |
| `0x0253` | length | referenced | `0x0253` | `3` | `-` | packet.go, taekwon_packets.go |
| `0x0254` | request | sender | `0x0254` | `3` | `clif_parse_FeelSaveOk` | taekwon_packets.go |
| `0x0255` | length | referenced | `0x0255` | `5` | `-` | mail_packets.go, packet.go |
| `0x0256` | length | untracked | `0x0256` | `5` | `-` | - |
| `0x0257` | length | referenced | `0x0257` | `8` | `-` | mail_packets.go, packet.go |
| `0x0258` | length | untracked | `0x0258` | `2` | `-` | - |
| `0x0259` | length | untracked | `0x0259` | `3` | `-` | - |
| `0x025B` | request | missing | `HEADER_CZ_REQ_MAKINGITEM` | `sizeof( struct PACKET_CZ_REQ_MAKINGITEM )` | `clif_parse_Cooking` | - |
| `0x025C` | request | missing | `0x025c` | `4` | `clif_parse_Auction_buysell` | - |
| `0x025D` | request | missing | `0x025d` | `6` | `clif_parse_Auction_close` | - |
| `0x025E` | length | untracked | `0x025e` | `4` | `-` | - |
| `0x025F` | length | untracked | `0x025f` | `6` | `-` | - |
| `0x0260` | length | referenced | `0x0260` | `6` | `-` | mail_packets.go, packet.go |
| `0x0261` | length | untracked | `0x0261` | `11` | `-` | - |
| `0x0262` | length | untracked | `0x0262` | `11` | `-` | - |
| `0x0263` | length | untracked | `0x0263` | `11` | `-` | - |
| `0x0264` | length | untracked | `0x0264` | `20` | `-` | - |
| `0x0265` | length | untracked | `0x0265` | `20` | `-` | - |
| `0x0266` | length | untracked | `0x0266` | `30` | `-` | - |
| `0x0267` | length | untracked | `0x0267` | `4` | `-` | - |
| `0x0268` | length | untracked | `0x0268` | `4` | `-` | - |
| `0x0269` | length | untracked | `0x0269` | `4` | `-` | - |
| `0x026A` | length | untracked | `0x026a` | `4` | `-` | - |
| `0x026B` | length | untracked | `0x026b` | `4` | `-` | - |
| `0x026C` | length | untracked | `0x026c` | `4` | `-` | - |
| `0x026D` | length | untracked | `0x026d` | `4` | `-` | - |
| `0x026F` | length | untracked | `0x026f` | `2` | `-` | - |
| `0x0270` | length | untracked | `0x0270` | `2` | `-` | - |
| `0x0271` | length | untracked | `0x0271` | `40` | `-` | - |
| `0x0272` | length | untracked | `0x0272` | `44` | `-` | - |
| `0x0273` | request | sender | `0x0273` | `30` | `clif_parse_Mail_return` | mail_packets.go |
| `0x0274` | length | referenced | `0x0274` | `8` | `-` | mail_packets.go, packet.go |
| `0x0277` | length | untracked | `0x0277` | `84` | `-` | - |
| `0x0278` | length | untracked | `0x0278` | `2` | `-` | - |
| `0x0279` | length | untracked | `0x0279` | `2` | `-` | - |
| `0x027A` | length | untracked | `0x027a` | `-1` | `-` | - |
| `0x027B` | length | untracked | `0x027b` | `14` | `-` | - |
| `0x027C` | length | untracked | `0x027c` | `60` | `-` | - |
| `0x027D` | length | referenced | `0x027d` | `62` | `-` | companion_packets.go, packet.go |
| `0x027E` | length | untracked | `0x027e` | `-1` | `-` | - |
| `0x027F` | length | untracked | `0x027f` | `8` | `-` | - |
| `0x0280` | length | untracked | `0x0280` | `12` | `-` | - |
| `0x0281` | length | untracked | `0x0281` | `4` | `-` | - |
| `0x0282` | length | untracked | `0x0282` | `284` | `-` | - |
| `0x0283` | length | referenced | `0x0283` | `6` | `-` | packet.go |
| `0x0284` | length | untracked | `0x0284` | `14` | `-` | - |
| `0x0285` | length | untracked | `0x0285` | `6` | `-` | - |
| `0x0286` | length | untracked | `0x0286` | `4` | `-` | - |
| `0x0287` | length | untracked | `0x0287` | `-1` | `-` | - |
| `0x0288` | request | missing | `0x0288` | `10` | `clif_parse_npccashshop_buy` | - |
| `0x0289` | length | untracked | `0x0289` | `12` | `-` | - |
| `0x028B` | length | untracked | `0x028b` | `-1` | `-` | - |
| `0x028C` | length | untracked | `0x028c` | `46` | `-` | - |
| `0x028D` | length | untracked | `0x028d` | `34` | `-` | - |
| `0x028E` | length | untracked | `0x028e` | `4` | `-` | - |
| `0x028F` | length | untracked | `0x028f` | `6` | `-` | - |
| `0x0290` | length | untracked | `0x0290` | `4` | `-` | - |
| `0x0292` | request | sender | `0x0292` | `2` | `clif_parse_AutoRevive` | revive_packets.go |
| `0x0293` | length | referenced | `0x0293` | `70` | `-` | packet.go, server_info_packets.go |
| `0x0294` | length | untracked | `0x0294` | `10` | `-` | - |
| `0x029B` | length | referenced | `0x029b` | `80` | `-` | companion_packets.go, packet.go |
| `0x029C` | length | referenced | `0x029c` | `66` | `-` | companion_packets.go, packet.go |
| `0x029D` | length | referenced | `0x029d` | `-1` | `-` | companion_packets.go, packet.go |
| `0x029E` | length | referenced | `0x029e` | `11` | `-` | companion_packets.go, packet.go |
| `0x029F` | request | sender | `0x029f` | `3` | `clif_parse_mercenary_action` | companion_packets.go |
| `0x02A0` | length | untracked | `0x02a0` | `-1` | `-` | - |
| `0x02A1` | length | untracked | `0x02a1` | `-1` | `-` | - |
| `0x02A2` | length | referenced | `0x02a2` | `8` | `-` | companion_packets.go, packet.go |
| `0x02A3` | length | untracked | `0x02a3` | `-1` | `-` | - |
| `0x02A4` | length | untracked | `0x02a4` | `-1` | `-` | - |
| `0x02A5` | length | untracked | `0x02a5` | `8` | `-` | - |
| `0x02A6` | length | untracked | `0x02a6` | `22` | `-` | - |
| `0x02A7` | length | untracked | `0x02a7` | `22` | `-` | - |
| `0x02A8` | length | untracked | `0x02a8` | `162` | `-` | - |
| `0x02A9` | length | untracked | `0x02a9` | `58` | `-` | - |
| `0x02AA` | length | untracked | `0x02aa` | `4` | `-` | - |
| `0x02AB` | length | untracked | `0x02ab` | `36` | `-` | - |
| `0x02AC` | length | untracked | `0x02ac` | `6` | `-` | - |
| `0x02AD` | length | untracked | `0x02ad` | `8` | `-` | - |
| `0x02B0` | length | referenced | `0x02b0` | `85` | `-` | packet.go |
| `0x02B1` | length | referenced | `0x02b1` | `-1` | `-` | packet.go, quest_packets.go |
| `0x02B2` | length | referenced | `0x02b2` | `-1` | `-` | packet.go, quest_packets.go |
| `0x02B3` | length | referenced | `0x02b3` | `107` | `-` | packet.go, quest_packets.go |
| `0x02B4` | length | referenced | `0x02b4` | `6` | `-` | packet.go, quest_packets.go |
| `0x02B5` | length | referenced | `0x02b5` | `-1` | `-` | packet.go, quest_packets.go |
| `0x02B6` | request | sender | `HEADER_CZ_ACTIVE_QUEST` | `sizeof( PACKET_CZ_ACTIVE_QUEST )` | `clif_parse_questStateAck` | quest_packets.go |
| `0x02B7` | length | referenced | `0x02b7` | `7` | `-` | packet.go, quest_packets.go |
| `0x02B9` | length | referenced | `0x02b9` | `191` | `-` | hotkey_packets.go, packet.go |
| `0x02BA` | request | sender | `0x02ba` | `11` | `clif_parse_Hotkey` | hotkey_packets.go |
| `0x02BC` | length | untracked | `0x02bc` | `6` | `-` | - |
| `0x02BF` | length | untracked | `0x02bf` | `-1` | `-` | - |
| `0x02C0` | length | untracked | `0x02c0` | `-1` | `-` | - |
| `0x02C1` | length | referenced | `0x02c1` | `-1` | `-` | chat_packets.go, packet.go |
| `0x02C2` | length | untracked | `0x02c2` | `-1` | `-` | - |
| `0x02C4` | request | sender | `HEADER_CZ_PARTY_JOIN_REQ` | `sizeof( PACKET_CZ_PARTY_JOIN_REQ )` | `clif_parse_PartyInvite2` | packet.go, party_packets.go |
| `0x02C5` | length | referenced | `0x02c5` | `30` | `-` | packet.go, party_packets.go |
| `0x02C7` | request | sender | `HEADER_CZ_PARTY_JOIN_REQ_ACK` | `sizeof( PACKET_CZ_PARTY_JOIN_REQ_ACK )` | `clif_parse_ReplyPartyInvite2` | packet.go, party_packets.go |
| `0x02C8` | request | sender | `HEADER_CZ_PARTY_CONFIG` | `sizeof( PACKET_CZ_PARTY_CONFIG )` | `clif_parse_PartyTick` | packet.go, party_packets.go |
| `0x02CA` | length | untracked | `0x02ca` | `3` | `-` | - |
| `0x02CB` | length | untracked | `0x02cb` | `65` | `-` | - |
| `0x02CC` | length | untracked | `0x02cc` | `4` | `-` | - |
| `0x02CD` | length | untracked | `0x02cd` | `71` | `-` | - |
| `0x02CE` | length | untracked | `0x02ce` | `10` | `-` | - |
| `0x02CF` | request | missing | `0x02cf` | `6` | `clif_parse_MemorialDungeonCommand` | - |
| `0x02D0` | length | referenced | `inventorylistequipType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02D1` | length | referenced | `storageListEquipType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02D2` | length | referenced | `cartlistequipType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02D5` | length | referenced | `0x02d5` | `2` | `-` | disconnect_packets.go, packet.go |
| `0x02D6` | request | sender | `0x02d6` | `6` | `clif_parse_ViewPlayerEquip` | equipment_packets.go, packet.go |
| `0x02D8` | request | sender | `0x02d8` | `10` | `clif_parse_configuration` | equipment_packets.go, packet.go |
| `0x02D9` | length | referenced | `0x02d9` | `10` | `-` | packet.go |
| `0x02DB` | request | missing | `0x02db` | `-1` | `clif_parse_BattleChat` | - |
| `0x02DC` | length | untracked | `0x02dc` | `-1` | `-` | - |
| `0x02DD` | length | referenced | `0x02dd` | `32` | `-` | packet.go |
| `0x02DE` | length | untracked | `0x02de` | `6` | `-` | - |
| `0x02DF` | length | untracked | `0x02df` | `36` | `-` | - |
| `0x02E0` | length | untracked | `0x02e0` | `34` | `-` | - |
| `0x02E2` | length | untracked | `0x02e2` | `14` | `-` | - |
| `0x02E3` | length | untracked | `0x02e3` | `25` | `-` | - |
| `0x02E4` | length | untracked | `0x02e4` | `8` | `-` | - |
| `0x02E5` | length | untracked | `0x02e5` | `8` | `-` | - |
| `0x02E6` | length | untracked | `0x02e6` | `6` | `-` | - |
| `0x02E7` | length | untracked | `0x02e7` | `-1` | `-` | - |
| `0x02E8` | length | referenced | `inventorylistnormalType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02E9` | length | referenced | `cartlistnormalType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02EA` | length | referenced | `storageListNormalType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02EC` | length | referenced | `0x02ec` | `67` | `-` | actor_packets.go, packet.go |
| `0x02ED` | length | referenced | `0x02ed` | `59` | `-` | actor_packets.go, packet.go |
| `0x02EE` | length | referenced | `0x02ee` | `60` | `-` | actor_packets.go, packet.go |
| `0x02EF` | length | referenced | `0x02ef` | `8` | `-` | packet.go |
| `0x02F0` | length | referenced | `0x02f0` | `10` | `-` | packet.go, server_info_packets.go |
| `0x02F1` | request | sender | `0x02f1` | `2` | `clif_parse_progressbar` | server_info_packets.go |
| `0x02F2` | length | referenced | `0x02f2` | `2` | `-` | packet.go, server_info_packets.go |
| `0x02F3` | length | untracked | `0x02f3` | `-1` | `-` | - |
| `0x02F4` | length | untracked | `0x02f4` | `-1` | `-` | - |
| `0x02F5` | length | untracked | `0x02f5` | `-1` | `-` | - |
| `0x02F6` | length | untracked | `0x02f6` | `-1` | `-` | - |
| `0x02F7` | length | untracked | `0x02f7` | `-1` | `-` | - |
| `0x02F8` | length | untracked | `0x02f8` | `-1` | `-` | - |
| `0x02F9` | length | untracked | `0x02f9` | `-1` | `-` | - |
| `0x02FA` | length | untracked | `0x02fa` | `-1` | `-` | - |
| `0x02FB` | length | untracked | `0x02fb` | `-1` | `-` | - |
| `0x02FC` | length | untracked | `0x02fc` | `-1` | `-` | - |
| `0x02FD` | length | untracked | `0x02fd` | `-1` | `-` | - |
| `0x02FE` | length | untracked | `0x02fe` | `-1` | `-` | - |
| `0x02FF` | length | untracked | `0x02ff` | `-1` | `-` | - |
| `0x0300` | length | untracked | `0x0300` | `-1` | `-` | - |
| `0x0301` | length | untracked | `0x0301` | `-1` | `-` | - |
| `0x0302` | length | untracked | `0x0302` | `-1` | `-` | - |
| `0x0303` | length | untracked | `0x0303` | `-1` | `-` | - |
| `0x0304` | length | untracked | `0x0304` | `-1` | `-` | - |
| `0x0305` | length | untracked | `0x0305` | `-1` | `-` | - |
| `0x0306` | length | untracked | `0x0306` | `-1` | `-` | - |
| `0x0307` | length | untracked | `0x0307` | `-1` | `-` | - |
| `0x0308` | length | untracked | `0x0308` | `-1` | `-` | - |
| `0x0309` | length | untracked | `0x0309` | `-1` | `-` | - |
| `0x030A` | length | untracked | `0x030a` | `-1` | `-` | - |
| `0x030B` | length | untracked | `0x030b` | `-1` | `-` | - |
| `0x030C` | length | untracked | `0x030c` | `-1` | `-` | - |
| `0x030D` | length | untracked | `0x030d` | `-1` | `-` | - |
| `0x030E` | length | untracked | `0x030e` | `-1` | `-` | - |
| `0x030F` | length | untracked | `0x030f` | `-1` | `-` | - |
| `0x0310` | length | untracked | `0x0310` | `-1` | `-` | - |
| `0x0311` | length | untracked | `0x0311` | `-1` | `-` | - |
| `0x0312` | length | untracked | `0x0312` | `-1` | `-` | - |
| `0x0313` | length | untracked | `0x0313` | `-1` | `-` | - |
| `0x0314` | length | untracked | `0x0314` | `-1` | `-` | - |
| `0x0315` | length | untracked | `0x0315` | `-1` | `-` | - |
| `0x0316` | length | untracked | `0x0316` | `-1` | `-` | - |
| `0x0317` | length | untracked | `0x0317` | `-1` | `-` | - |
| `0x0318` | length | untracked | `0x0318` | `-1` | `-` | - |
| `0x0319` | length | untracked | `0x0319` | `-1` | `-` | - |
| `0x031A` | length | untracked | `0x031a` | `-1` | `-` | - |
| `0x031B` | length | untracked | `0x031b` | `-1` | `-` | - |
| `0x031C` | length | untracked | `0x031c` | `-1` | `-` | - |
| `0x031D` | length | untracked | `0x031d` | `-1` | `-` | - |
| `0x031E` | length | untracked | `0x031e` | `-1` | `-` | - |
| `0x031F` | length | untracked | `0x031f` | `-1` | `-` | - |
| `0x0320` | length | untracked | `0x0320` | `-1` | `-` | - |
| `0x0321` | length | untracked | `0x0321` | `-1` | `-` | - |
| `0x0322` | length | untracked | `0x0322` | `-1` | `-` | - |
| `0x0323` | length | untracked | `0x0323` | `-1` | `-` | - |
| `0x0324` | length | untracked | `0x0324` | `-1` | `-` | - |
| `0x0325` | length | untracked | `0x0325` | `-1` | `-` | - |
| `0x0326` | length | untracked | `0x0326` | `-1` | `-` | - |
| `0x0327` | length | untracked | `0x0327` | `-1` | `-` | - |
| `0x0328` | length | untracked | `0x0328` | `-1` | `-` | - |
| `0x0329` | length | untracked | `0x0329` | `-1` | `-` | - |
| `0x032A` | length | untracked | `0x032a` | `-1` | `-` | - |
| `0x032B` | length | untracked | `0x032b` | `-1` | `-` | - |
| `0x032C` | length | untracked | `0x032c` | `-1` | `-` | - |
| `0x032D` | length | untracked | `0x032d` | `-1` | `-` | - |
| `0x032E` | length | untracked | `0x032e` | `-1` | `-` | - |
| `0x032F` | length | untracked | `0x032f` | `-1` | `-` | - |
| `0x0330` | length | untracked | `0x0330` | `-1` | `-` | - |
| `0x0331` | length | untracked | `0x0331` | `-1` | `-` | - |
| `0x0332` | length | untracked | `0x0332` | `-1` | `-` | - |
| `0x0333` | length | untracked | `0x0333` | `-1` | `-` | - |
| `0x0334` | length | untracked | `0x0334` | `-1` | `-` | - |
| `0x0335` | length | untracked | `0x0335` | `-1` | `-` | - |
| `0x0336` | length | untracked | `0x0336` | `-1` | `-` | - |
| `0x0337` | length | untracked | `0x0337` | `-1` | `-` | - |
| `0x0338` | length | untracked | `0x0338` | `-1` | `-` | - |
| `0x0339` | length | untracked | `0x0339` | `-1` | `-` | - |
| `0x033A` | length | untracked | `0x033a` | `-1` | `-` | - |
| `0x033B` | length | untracked | `0x033b` | `-1` | `-` | - |
| `0x033C` | length | untracked | `0x033c` | `-1` | `-` | - |
| `0x033D` | length | untracked | `0x033d` | `-1` | `-` | - |
| `0x033E` | length | untracked | `0x033e` | `-1` | `-` | - |
| `0x033F` | length | untracked | `0x033f` | `-1` | `-` | - |
| `0x0340` | length | untracked | `0x0340` | `-1` | `-` | - |
| `0x0341` | length | untracked | `0x0341` | `-1` | `-` | - |
| `0x0342` | length | untracked | `0x0342` | `-1` | `-` | - |
| `0x0343` | length | untracked | `0x0343` | `-1` | `-` | - |
| `0x0344` | length | untracked | `0x0344` | `-1` | `-` | - |
| `0x0345` | length | untracked | `0x0345` | `-1` | `-` | - |
| `0x0346` | length | untracked | `0x0346` | `-1` | `-` | - |
| `0x0347` | length | untracked | `0x0347` | `-1` | `-` | - |
| `0x0348` | length | untracked | `0x0348` | `-1` | `-` | - |
| `0x0349` | length | untracked | `0x0349` | `-1` | `-` | - |
| `0x034A` | length | untracked | `0x034a` | `-1` | `-` | - |
| `0x034B` | length | untracked | `0x034b` | `-1` | `-` | - |
| `0x034C` | length | untracked | `0x034c` | `-1` | `-` | - |
| `0x034D` | length | untracked | `0x034d` | `-1` | `-` | - |
| `0x034E` | length | untracked | `0x034e` | `-1` | `-` | - |
| `0x034F` | length | untracked | `0x034f` | `-1` | `-` | - |
| `0x0350` | length | untracked | `0x0350` | `-1` | `-` | - |
| `0x0351` | length | untracked | `0x0351` | `-1` | `-` | - |
| `0x0352` | length | untracked | `0x0352` | `-1` | `-` | - |
| `0x0353` | length | untracked | `0x0353` | `-1` | `-` | - |
| `0x0354` | length | untracked | `0x0354` | `-1` | `-` | - |
| `0x0355` | length | untracked | `0x0355` | `-1` | `-` | - |
| `0x0356` | length | untracked | `0x0356` | `-1` | `-` | - |
| `0x0357` | length | untracked | `0x0357` | `-1` | `-` | - |
| `0x0358` | length | untracked | `0x0358` | `-1` | `-` | - |
| `0x0359` | length | untracked | `0x0359` | `-1` | `-` | - |
| `0x035A` | length | untracked | `0x035a` | `-1` | `-` | - |
| `0x035B` | length | untracked | `0x035b` | `-1` | `-` | - |
| `0x035C` | length | untracked | `0x035c` | `2` | `-` | - |
| `0x035D` | length | untracked | `0x035d` | `-1` | `-` | - |
| `0x035E` | length | untracked | `0x035e` | `2` | `-` | - |
| `0x035F` | length | referenced | `0x035f` | `-1` | `-` | login_packets.go |
| `0x0389` | length | untracked | `0x0389` | `-1` | `-` | - |
| `0x040C` | length | untracked | `0x040c` | `-1` | `-` | - |
| `0x040D` | length | untracked | `0x040d` | `-1` | `-` | - |
| `0x040E` | length | untracked | `0x040e` | `-1` | `-` | - |
| `0x040F` | length | untracked | `0x040f` | `-1` | `-` | - |
| `0x0410` | length | untracked | `0x0410` | `-1` | `-` | - |
| `0x0411` | length | untracked | `0x0411` | `-1` | `-` | - |
| `0x0412` | length | untracked | `0x0412` | `-1` | `-` | - |
| `0x0413` | length | untracked | `0x0413` | `-1` | `-` | - |
| `0x0414` | length | untracked | `0x0414` | `-1` | `-` | - |
| `0x0415` | length | untracked | `0x0415` | `-1` | `-` | - |
| `0x0416` | length | untracked | `0x0416` | `-1` | `-` | - |
| `0x0417` | length | untracked | `0x0417` | `-1` | `-` | - |
| `0x0418` | length | untracked | `0x0418` | `-1` | `-` | - |
| `0x0419` | length | untracked | `0x0419` | `-1` | `-` | - |
| `0x041A` | length | untracked | `0x041a` | `-1` | `-` | - |
| `0x041B` | length | untracked | `0x041b` | `-1` | `-` | - |
| `0x041C` | length | untracked | `0x041c` | `-1` | `-` | - |
| `0x041D` | length | untracked | `0x041d` | `-1` | `-` | - |
| `0x041E` | length | untracked | `0x041e` | `-1` | `-` | - |
| `0x041F` | length | untracked | `0x041f` | `-1` | `-` | - |
| `0x0420` | length | untracked | `0x0420` | `-1` | `-` | - |
| `0x0421` | length | untracked | `0x0421` | `-1` | `-` | - |
| `0x0422` | length | untracked | `0x0422` | `-1` | `-` | - |
| `0x0423` | length | untracked | `0x0423` | `-1` | `-` | - |
| `0x0424` | length | untracked | `0x0424` | `-1` | `-` | - |
| `0x0425` | length | untracked | `0x0425` | `-1` | `-` | - |
| `0x0426` | length | untracked | `0x0426` | `-1` | `-` | - |
| `0x0427` | length | untracked | `0x0427` | `-1` | `-` | - |
| `0x0428` | length | untracked | `0x0428` | `-1` | `-` | - |
| `0x0429` | length | untracked | `0x0429` | `-1` | `-` | - |
| `0x042A` | length | untracked | `0x042a` | `-1` | `-` | - |
| `0x042B` | length | untracked | `0x042b` | `-1` | `-` | - |
| `0x042C` | length | untracked | `0x042c` | `-1` | `-` | - |
| `0x042D` | length | untracked | `0x042d` | `-1` | `-` | - |
| `0x042E` | length | untracked | `0x042e` | `-1` | `-` | - |
| `0x042F` | length | untracked | `0x042f` | `-1` | `-` | - |
| `0x0430` | length | untracked | `0x0430` | `-1` | `-` | - |
| `0x0431` | length | untracked | `0x0431` | `-1` | `-` | - |
| `0x0432` | length | untracked | `0x0432` | `-1` | `-` | - |
| `0x0433` | length | untracked | `0x0433` | `-1` | `-` | - |
| `0x0434` | length | untracked | `0x0434` | `-1` | `-` | - |
| `0x0435` | length | untracked | `0x0435` | `-1` | `-` | - |
| `0x0436` | request | sender | `0x0436` | `19` | `clif_parse_WantToConnection` | login_packets.go |
| `0x0437` | request | alias-covered | `0x0437` | `7` | `clif_parse_ActionRequest` | login_packets.go |
| `0x0438` | request | sender | `0x0438` | `10` | `clif_parse_UseSkillToId` | skill_packets.go |
| `0x0439` | request | sender | `0x0439` | `8` | `clif_parse_UseItem` | item_packets.go |
| `0x0802` | request | post-2008 | `0x0802` | `18` | `clif_parse_PartyBookingRegisterReq` | - |
| `0x0803` | length | untracked | `0x0803` | `4` | `-` | - |
| `0x0804` | length | untracked | `0x0804` | `8` | `-` | - |
| `0x0805` | length | untracked | `0x0805` | `-1` | `-` | - |
| `0x0806` | request | post-2008 | `0x0806` | `4` | `clif_parse_PartyBookingDeleteReq` | - |
| `0x0808` | length | untracked | `0x0808` | `4` | `-` | - |
| `0x08B3` | length | untracked | `0x8b3` | `-1` | `-` | - |

## Maintenance notes

- Regenerate the effective table when changing the server profile or packet DB.
- Recount opcode references and actual 2008 sender paths separately. An opcode
  used for a different date/action is not a valid sender for the winning handler.
- Keep full lifecycle tests and manual server checks separate from these counts.
- Do not add server-to-client header declarations to the effective DB table;
  many valid messages are outside it and need a separate receive-side audit.
