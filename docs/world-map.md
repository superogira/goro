# World map (2008)

Press **Ctrl + the physical key immediately left of 1** (the US backtick/tilde
key, also usable on AZERTY). The same shortcut, Escape, or the close button
closes the map. The magnifying glass in its title bar toggles map names.

The map uses `worldmap.bmp` and `mapPosTable.txt` from the client data. The
illustration keeps its aspect ratio and fits the screen. A star marks your
current map; online party members highlight their maps. Hover a field/town to
see its minimap and the names of party members there. Maps absent from the
original table, including many interiors and dungeons, have no invented marker.

The preview shows player and party coordinates on the current map only:
rAthena sends `0x0107` party-position updates to same-map members. Remote party
members' map names are known, but their last coordinates must not be reused as
if they were current. Missing minimap images are handled explicitly.

This is an atlas, not a navigation tool. No route planning, click-to-walk,
monster search, quest markers, or later regional map selectors are included.
The existing Map button still toggles the small HUD minimap.

Artwork and previews (including missing images) are cached. Closed windows do
no presentation work. The visible atlas only redraws when displayed state
changes; walking does not redraw it unless the current-map preview is visible.

## References

- `2008-09-10aSakexe.exe`: references `worldmap.bmp`, `mapPosTable.txt`, the Ctrl+backtick shortcut, and the `world_view` controls.
- [Original December 2007 kRO patch notes, reproduced](https://ragnarok.fandom.com/wiki/RO_Patch_%282007_Dec._05%29): shortcut, map-name toggle, party hover information, and player star.
- [Official December 2008 iRO update](https://renewal.playragnarok.com/news/updatedetail.aspx?id=80): world-map introduction, shortcut, player and party locations.
- rAthena `clif_party_xy`: exact party coordinates are sent with `PARTY_SAMEMAP_WOS`.

## Manual checks

1. Open from a town, check the star, hover nearby fields, and toggle names.
2. Click and right-click the map: neither should move the character underneath.
3. Close with Escape and reopen with the shortcut, including on AZERTY.
4. With a party, check same-map dots and remote-map highlights/names; then have a member move maps or log out.
5. Warp to another town with the window open; check the star and preview refresh.
6. Enter an unlisted interior: it must not retain the previous town's star.
