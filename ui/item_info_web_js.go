//go:build js && wasm

package ui

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

// DOM twin of upstream's ItemWindows: item descriptions and card artwork
// open as independent, individually closable page panels. Each description
// snapshots the item, so inspecting another item never rewrites an open
// one; clicking a slotted card opens that card's own description, and a
// card description's View button opens its artwork panel.

const (
	itemInfoWebKindDesc = "desc"
	itemInfoWebKindArt  = "art"
)

type itemInfoWebWindow struct {
	id        int
	seq       int // creation index; the page cascades initial positions from it
	kind      string
	item      session.InventoryItem
	title     string
	artURL    string
	viewable  bool
	contentJS js.Value
}

var itemInfoWebState = struct {
	windows []*itemInfoWebWindow
	nextID  int
	nextSeq int
	syncKey string
	hooked  bool
}{}

func itemInfoWebEnabled() bool {
	return js.Global().Get("goroItemInfoWindowsSync").Type() == js.TypeFunction
}

// itemInfoWebShow opens (or appends) a DOM item description panel. Shared
// entry point for the inventory and equipment windows.
func itemInfoWebShow(ctx Context, item session.InventoryItem) {
	itemInfoWebOpenDesc(ctx, item)
}

func itemInfoWebOpenDesc(ctx Context, item session.InventoryItem) {
	if item.ItemID == 0 || !itemInfoWebEnabled() {
		return
	}
	w := &itemInfoWebWindow{
		id:   itemInfoWebState.nextID,
		seq:  itemInfoWebState.nextSeq,
		kind: itemInfoWebKindDesc,
		item: item,
	}
	itemInfoWebState.nextID++
	itemInfoWebState.nextSeq++
	w.title = inventoryItemDisplayName(ctx.Resources, item)
	if item.Refine > 0 {
		w.title = fmt.Sprintf("+%d %s", item.Refine, w.title)
	}
	if item.Type == db.ItemTypeCard {
		if _, ok := ctx.Resources.ItemCardIllustrationName(int(item.ItemID)); ok {
			w.viewable = true
		}
	}
	w.contentJS = itemInfoWebDescPayload(ctx, item, w.title)
	itemInfoWebState.windows = append(itemInfoWebState.windows, w)
}

func itemInfoWebOpenArtwork(ctx Context, cardID uint16, title string) {
	if cardID == 0 || ctx.Resources == nil || !itemInfoWebEnabled() {
		return
	}
	resource, ok := ctx.Resources.ItemCardIllustrationName(int(cardID))
	if !ok {
		return
	}
	key := fmt.Sprintf("cardart:%d", cardID)
	url := hotbarWebIconCache[key]
	if url == "" {
		img, _, err := res.LoadImage(ctx.Resources, res.CardIllustrationTextureCandidates(resource))
		if err != nil {
			return
		}
		url = hotbarWebIcon(key, img)
	}
	if url == "" {
		return
	}
	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("Card #%d", cardID)
	}
	w := &itemInfoWebWindow{
		id:      itemInfoWebState.nextID,
		seq:     itemInfoWebState.nextSeq,
		kind:    itemInfoWebKindArt,
		title:   title,
		artURL:  url,
		contentJS: func() js.Value {
			obj := js.Global().Get("Object").New()
			obj.Set("kind", itemInfoWebKindArt)
			obj.Set("title", title)
			obj.Set("art", url)
			return obj
		}(),
	}
	itemInfoWebState.nextID++
	itemInfoWebState.nextSeq++
	itemInfoWebState.windows = append(itemInfoWebState.windows, w)
}

// UpdateItemInfoWebWindows drains page interactions, owns Escape (closes
// the topmost panel, mirroring upstream's window stack), and publishes the
// window set to the page. Consumed returns true only when this update ate
// the Escape press.
func UpdateItemInfoWebWindows(ctx client.Context) bool {
	if !itemInfoWebEnabled() || ctx.Input == nil {
		return false
	}
	itemInfoWebInstallHooks()
	for _, action := range hudWebDrainActions("iteminfo:") {
		parts := strings.Split(strings.TrimPrefix(action, "iteminfo:"), ":")
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "close":
			if len(parts) >= 2 {
				if id, err := strconv.Atoi(parts[1]); err == nil {
					itemInfoWebRemove(id)
				}
			}
		case "card":
			if len(parts) >= 3 {
				id, err1 := strconv.Atoi(parts[1])
				slot, err2 := strconv.Atoi(parts[2])
				if err1 != nil || err2 != nil {
					continue
				}
				if w := itemInfoWebFind(id); w != nil && slot < len(w.item.Cards) {
					cardID := w.item.Cards[slot]
					if cardID == 0x00ff || cardID == 0x00fe || cardID == 0xff00 {
						cardID = 0
					}
					if cardID != 0 {
						itemInfoWebOpenDesc(ctx, session.InventoryItem{ItemID: cardID, Type: db.ItemTypeCard, Identified: true})
					}
				}
			}
		case "view":
			if len(parts) >= 2 {
				if id, err := strconv.Atoi(parts[1]); err == nil {
					if w := itemInfoWebFind(id); w != nil && w.viewable {
						itemInfoWebOpenArtwork(ctx, w.item.ItemID, w.title)
					}
				}
			}
		}
	}
	consumed := false
	if ctx.Input.JustPressed(input.KeyEscape) && len(itemInfoWebState.windows) > 0 {
		// The page owns stacking order (raise on press); let it close its
		// own topmost panel and drop that instance here.
		if fn := js.Global().Get("goroItemInfoEscape"); fn.Type() == js.TypeFunction {
			if id := fn.Invoke().Int(); id >= 0 {
				itemInfoWebRemove(id)
			}
		}
		consumed = true
	}
	itemInfoWebSync()
	return consumed
}

func itemInfoWebInstallHooks() {
	if itemInfoWebState.hooked {
		return
	}
	itemInfoWebState.hooked = true
	js.Global().Set("goroItemInfoAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "iteminfo:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
}

func itemInfoWebFind(id int) *itemInfoWebWindow {
	for _, w := range itemInfoWebState.windows {
		if w.id == id {
			return w
		}
	}
	return nil
}

func itemInfoWebRemove(id int) {
	for i, w := range itemInfoWebState.windows {
		if w.id == id {
			itemInfoWebState.windows = append(itemInfoWebState.windows[:i], itemInfoWebState.windows[i+1:]...)
			return
		}
	}
}

func itemInfoWebSync() {
	key := ""
	for _, w := range itemInfoWebState.windows {
		key += fmt.Sprintf("%d:%s;", w.id, w.kind)
	}
	if key == itemInfoWebState.syncKey {
		return
	}
	itemInfoWebState.syncKey = key
	list := js.Global().Get("Array").New(len(itemInfoWebState.windows))
	for i, w := range itemInfoWebState.windows {
		obj := w.contentJS
		obj.Set("id", w.id)
		obj.Set("seq", w.seq)
		obj.Set("viewable", w.viewable)
		list.SetIndex(i, obj)
	}
	js.Global().Get("goroItemInfoWindowsSync").Invoke(list)
}

// itemInfoWebDescPayload builds the description panel payload: title,
// description lines, card names, card-slot footer, and the collection
// illustration as a cached data URL.
func itemInfoWebDescPayload(ctx Context, item session.InventoryItem, title string) js.Value {
	obj := js.Global().Get("Object").New()
	obj.Set("kind", itemInfoWebKindDesc)
	obj.Set("title", title)
	desc, _ := ctx.Resources.ItemDescription(int(item.ItemID), item.Identified)
	obj.Set("descHTML", itemInfoWebDescHTML(strings.Join(desc, "\n")))
	cards := js.Global().Get("Array").New(0)
	for _, cardID := range item.Cards {
		if cardID == 0 {
			continue
		}
		if name, ok := ctx.Resources.ItemDisplayName(int(cardID), true); ok {
			cards.Call("push", name)
		}
	}
	obj.Set("cards", cards)
	slotCount, _ := ctx.Resources.ItemSlotCount(int(item.ItemID))
	if item.Type == db.ItemTypeWeapon || item.Type == db.ItemTypeArmor || item.Type == db.ItemTypeShadowGear {
		slots := js.Global().Get("Array").New(slotCount)
		for i := 0; i < slotCount; i++ {
			entry := js.Global().Get("Object").New()
			cardID := uint16(0)
			if i < len(item.Cards) {
				cardID = item.Cards[i]
				if cardID == 0x00ff || cardID == 0x00fe || cardID == 0xff00 {
					cardID = 0
				}
			}
			if cardID != 0 {
				entry.Set("state", "card")
				if name, ok := ctx.Resources.ItemDisplayName(int(cardID), true); ok {
					entry.Set("name", name)
				}
				if cardRes, ok := ctx.Resources.ItemResourceName(int(cardID), true); ok {
					entry.Set("icon", collectionWebIcon(ctx, cardRes, cardID, true))
				}
			} else {
				entry.Set("state", "empty")
			}
			slots.SetIndex(i, entry)
		}
		obj.Set("slots", slots)
	}
	if resourceName, ok := ctx.Resources.ItemResourceName(int(item.ItemID), item.Identified); ok {
		obj.Set("icon", collectionWebIcon(ctx, resourceName, item.ItemID, item.Identified))
	} else {
		obj.Set("icon", "")
	}
	return obj
}
