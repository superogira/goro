# Quest journal (2008)

`Alt+U` toggles the quest journal. Its left tabs show Active, Inactive, or All
quests. Select a quest and click View (or double-click it) for its description,
hunting progress, and time limit. Right-click a quest to request activation or
deactivation; the journal waits for the server's reply. Inactive does not mean
abandoned.

Only quests registered with the server's quest system appear here. Older NPC
scripts that use character variables alone do not automatically become journal
entries. Completing or erasing a quest removes it: the legacy protocol does not
send a completed-history list. A time limit expiring does not remove a quest
locally; the NPC/server still decides its outcome.

Titles, descriptions, and artwork come from `data/questid2display.txt` in the
client data. Unlisted server quests retain their ID, objectives, and deadline,
with a `Quest #ID` title.

Quest NPC markers and automatic quest/objective minimap overlays are outside our
2008 scope, not a planned follow-up. rAthena's `clif_quest_show_event` gates the
NPC marker packet (`0x0446`) on `PACKETVER >= 20090218`. Existing server compass
markers are a separate feature and remain supported.

## Testing on local rAthena

Use a spare GM character so these commands do not affect a real quest chain:

1. `@setquest 1001`, then `Alt+U`: the OldRO data includes the Acolyte quest's
   description. Open its details, right-click its row, and check all three tabs.
2. Change maps and relog: its state should survive. `@completequest 1001` should
   remove it, including an open detail window. `@erasequest 1001` clears the
   completed record afterward.
3. `@setquest 60107`: the local pre-renewal database defines a 50-Fabre hunt.
   Kill a Fabre and check the counter. The older data may not name this quest;
   the fallback title is expected. `@erasequest 60107` cleans it up.
4. `@setquest 2143` starts a 50-second timer in the local database. Leave its
   details open to check expiry, then clean up with `@erasequest 2143`.

Do not confuse a finished kill counter with quest completion: the NPC normally
checks it and completes the quest when handing in. Deadline and zero-counter
updates are also covered by automated tests.

## Implementation references

Checked against roBrowser's `QuestV1`, classic-ro-client's quest events/display
table, the 2008 Sakexe packet dispatch, and rAthena's `clif_quest_*` functions:

- `0x02B1`: available quest IDs and active flags; replaces the session snapshot.
- `0x02B2`: missions, expiry timestamps, and current counts. Every legacy record
  reserves three objective slots, regardless of its objective count.
- `0x02B3`: add/replace one quest, also with three reserved slots.
- `0x02B4`: erase or complete a quest.
- `0x02B5`: hunting counters, including required totals absent from `0x02B2/3`.
- `0x02B6` / `0x02B7`: activation request / server acknowledgement.

Quest state belongs to the character session and survives map changes. Windows
reuse the existing tabs, table, scroll view, and window lifecycle. Metadata and
artwork are cached; journal contents rebuild only when their state changes,
and the detail countdown updates once per second without rebuilding the window.
