package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
	worldstate "github.com/kivutar/goro/world"
)

func TestShopBuyAndSellTotalsAlignWithFooterButtons(t *testing.T) {
	for _, mode := range []int{shopModeBuy, shopModeSell} {
		window := ShopWindow{mode: mode}
		root := window.buyCartWidgetTree(Context{})
		root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(shopBuyCartWindowW, float32(window.cartWindowHeight()))))

		var total widget.Widget
		var walk func(widget.Widget)
		walk = func(current widget.Widget) {
			if total != nil {
				return
			}
			if text, ok := current.(interface{ Content() string }); ok && strings.HasPrefix(text.Content(), "Total:") {
				total = current
				return
			}
			for _, child := range current.Children() {
				walk(child)
			}
		}
		walk(root)
		if total == nil {
			t.Fatalf("shop mode %d total label not found", mode)
		}
		bounds := total.(interface{ Bounds() geometry.Rect }).Bounds()
		if bounds.Min.Y != rotheme.ButtonPaddingY {
			t.Fatalf("shop mode %d total label y = %.1f, want %.1f to align with footer buttons", mode, bounds.Min.Y, rotheme.ButtonPaddingY)
		}
		parent := total.(interface{ Parent() widget.Widget }).Parent()
		parentBounds := parent.(interface{ Bounds() geometry.Rect }).Bounds()
		wantHeight := rotheme.Default.Typography.TextSize + rotheme.ButtonPaddingY*2
		if parentBounds.Height() != wantHeight {
			t.Fatalf("shop mode %d total wrapper height = %.1f, want %.1f", mode, parentBounds.Height(), wantHeight)
		}
	}
}

func TestShopAddSellCartItemTracksAmount(t *testing.T) {
	window := ShopWindow{
		mode: shopModeSell,
	}

	window.addCartItem(session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, network.ShopSellItem{Index: 7, Price: 10, OverchargePrice: 12})
	window.addCartItem(session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, network.ShopSellItem{Index: 7, Price: 10, OverchargePrice: 12})
	if len(window.cart) != 1 || window.cart[0].amount != 3 || window.cart[0].max != 3 || window.cart[0].over != 12 {
		t.Fatalf("cart = %+v", window.cart)
	}
}

func TestShopBuyCartTracksQuantityAndTotal(t *testing.T) {
	window := ShopWindow{mode: shopModeBuy}
	item := network.ShopBuyItem{ItemID: 501, Price: 100, DiscountPrice: 80}

	window.addBuyItem(item)
	window.addBuyItem(item)
	if got := window.buyCart[0].amount; got != 2 {
		t.Fatalf("buy amount = %d, want 2", got)
	}
	if got := window.total(); got != 160 {
		t.Fatalf("total = %d, want 160", got)
	}

	window.decrementBuyCartRow(0)
	if got := window.buyCart[0].amount; got != 1 {
		t.Fatalf("buy amount after decrement = %d, want 1", got)
	}
}

func TestShopBuyStackableItemUsesAmountPrompt(t *testing.T) {
	window := ShopWindow{
		mode:     shopModeBuy,
		buyItems: []network.ShopBuyItem{{ItemID: 501, Type: db.ItemTypeHealing, Price: 100}},
	}
	ctx := Context{ScreenW: 800, ScreenH: 600}

	window.transferShopRowToCart(ctx, 0)
	if !window.amountPrompt.IsOpen() {
		t.Fatal("stackable shop item should open amount prompt")
	}
	if len(window.buyCart) != 0 {
		t.Fatalf("buy cart changed before amount submit: %+v", window.buyCart)
	}

	window.amountPrompt.value = "7"
	window.amountPrompt.submit(ctx)
	if len(window.buyCart) != 1 || window.buyCart[0].amount != 7 {
		t.Fatalf("buy cart = %+v, want one item amount 7", window.buyCart)
	}
}

func TestShopWindowBlocksKeyboardShortcutsDuringTransaction(t *testing.T) {
	window := ShopWindow{}
	if window.KeyboardShortcutsBlocked() {
		t.Fatal("closed shop should not block keyboard shortcuts")
	}

	ctx := Context{ScreenW: 800, ScreenH: 600}
	window.OpenBuy([]network.ShopBuyItem{{ItemID: 501, Type: db.ItemTypeHealing, Price: 100}}, ctx)
	if !window.KeyboardShortcutsBlocked() {
		t.Fatal("open buy shop should block keyboard shortcuts")
	}

	window.transferShopRowToCart(ctx, 0)
	if !window.KeyboardShortcutsBlocked() {
		t.Fatal("shop amount prompt should block keyboard shortcuts")
	}

	window.ApplyResult(ctx, network.ShopResult{Sell: false, Result: 0})
	if window.KeyboardShortcutsBlocked() {
		t.Fatal("completed shop should stop blocking keyboard shortcuts")
	}
}

func TestShopAmountPromptStaysAboveShopWindowsAfterRefresh(t *testing.T) {
	manager := NewManager()
	window := ShopWindow{}
	ctx := Context{ScreenW: 800, ScreenH: 600, UIManager: manager}
	window.OpenBuy([]network.ShopBuyItem{{ItemID: 501, Type: db.ItemTypeHealing, Price: 100}}, ctx)

	window.transferShopRowToCart(ctx, 0)
	window.refreshBuyWindow(ctx)

	if !window.amountPrompt.IsOpen() {
		t.Fatal("amount prompt should be open")
	}
	if len(manager.overlays) == 0 {
		t.Fatal("no overlays published")
	}
	if got := manager.overlays[len(manager.overlays)-1]; got != window.amountPrompt.published {
		t.Fatalf("top overlay = %T, want amount prompt", got)
	}
}

func TestShopBuyNonStackableItemSkipsAmountPrompt(t *testing.T) {
	window := ShopWindow{
		mode:     shopModeBuy,
		buyItems: []network.ShopBuyItem{{ItemID: 1201, Type: db.ItemTypeWeapon, Price: 100}},
	}

	window.transferShopRowToCart(Context{ScreenW: 800, ScreenH: 600}, 0)
	if window.amountPrompt.IsOpen() {
		t.Fatal("non-stackable shop item should not open amount prompt")
	}
	if len(window.buyCart) != 1 || window.buyCart[0].amount != 1 {
		t.Fatalf("buy cart = %+v, want one item amount 1", window.buyCart)
	}
}

func TestShopSellStackableItemUsesAmountPrompt(t *testing.T) {
	window := ShopWindow{
		mode:     shopModeSell,
		sellable: map[uint16]network.ShopSellItem{8: {Index: 8, Price: 10, OverchargePrice: 12}},
	}
	ctx := Context{
		ScreenW: 800,
		ScreenH: 600,
		Session: &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
			{Index: 8, ItemID: 938, Type: db.ItemTypeEtc, Amount: 9},
		}}},
	}

	window.transferShopRowToCart(ctx, 0)
	if !window.amountPrompt.IsOpen() {
		t.Fatal("stackable sell item should open amount prompt")
	}

	window.amountPrompt.value = "4"
	window.amountPrompt.submit(ctx)
	if len(window.cart) != 1 || window.cart[0].amount != 4 || window.cart[0].max != 9 {
		t.Fatalf("sell cart = %+v, want one item amount 4 max 9", window.cart)
	}
	available := window.sellAvailableItems(ctx)
	if len(available) != 1 || available[0].Index != 8 || available[0].Amount != 5 {
		t.Fatalf("available sell items = %+v, want index 8 amount 5", available)
	}
}

func TestShopSellAvailableItemsExcludeFullyStagedItems(t *testing.T) {
	window := ShopWindow{
		mode: shopModeSell,
		sellable: map[uint16]network.ShopSellItem{
			8: {Index: 8, Price: 10},
			9: {Index: 9, Price: 20},
		},
		cart: []shopSellCartItem{
			{item: session.InventoryItem{Index: 8, ItemID: 938, Type: db.ItemTypeEtc, Amount: 9}, amount: 9, max: 9},
		},
	}
	ctx := Context{Session: &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
		{Index: 8, ItemID: 938, Type: db.ItemTypeEtc, Amount: 9},
		{Index: 9, ItemID: 1201, Type: db.ItemTypeWeapon, Amount: 1},
	}}}}

	available := window.sellAvailableItems(ctx)
	if len(available) != 1 || available[0].Index != 9 {
		t.Fatalf("available sell items = %+v, want only index 9", available)
	}

	window.decrementSellCartRow(0)
	available = window.sellAvailableItems(ctx)
	if len(available) != 2 || available[0].Index != 8 || available[0].Amount != 9 || available[1].Index != 9 {
		t.Fatalf("available sell items after removal = %+v, want indexes 8 and 9", available)
	}
}

func TestShopRemoveBuyCartStackUsesAmountPrompt(t *testing.T) {
	window := ShopWindow{
		mode: shopModeBuy,
		buyCart: []shopBuyCartItem{{
			item:   network.ShopBuyItem{ItemID: 501, Type: db.ItemTypeHealing, Price: 100},
			amount: 8,
		}},
	}
	ctx := Context{ScreenW: 800, ScreenH: 600}

	window.transferCartRowToShop(ctx, 0)
	if !window.amountPrompt.IsOpen() {
		t.Fatal("stackable cart item should open amount prompt before removal")
	}

	window.amountPrompt.value = "3"
	window.amountPrompt.submit(ctx)
	if len(window.buyCart) != 1 || window.buyCart[0].amount != 5 {
		t.Fatalf("buy cart = %+v, want remaining amount 5", window.buyCart)
	}
}

func TestShopItemAtUsesTableViewBody(t *testing.T) {
	window := ShopWindow{
		mode:     shopModeBuy,
		buyItems: []network.ShopBuyItem{{ItemID: 501, Type: 0, Price: 100}},
	}
	window.buyWindow = NewWindow(shopBuyListWindowW, shopListWindowHeight())
	window.buyWindow.OpenAt(80, 90, nil)
	ctx := Context{}

	if _, ok := window.shopItemAt(ctx, window.buyWindow.x+8, window.buyWindow.y+ROWindowTitleHeight+shopTableHeaderH-1); ok {
		t.Fatal("shop header should not hit an item row")
	}

	item, ok := window.shopItemAt(ctx, window.buyWindow.x+8, window.buyWindow.y+ROWindowTitleHeight+shopTableHeaderH+1)
	if !ok {
		t.Fatal("shop row at top of table was not found")
	}
	if item.ItemID != 501 || item.Amount != 1 {
		t.Fatalf("shop item = %+v, want item 501 amount 1", item)
	}
}

func TestInventoryBagClassifiesTabs(t *testing.T) {
	tests := []struct {
		name string
		item session.InventoryItem
		tab  int
	}{
		{name: "healing item", item: session.InventoryItem{Type: 0}, tab: inventoryBagTabItem},
		{name: "usable item", item: session.InventoryItem{Type: 2}, tab: inventoryBagTabItem},
		{name: "equipment flag", item: session.InventoryItem{Type: 4, Equip: true}, tab: inventoryBagTabEquip},
		{name: "weapon type", item: session.InventoryItem{Type: 5}, tab: inventoryBagTabEquip},
		{name: "pet egg type", item: session.InventoryItem{Type: 7}, tab: inventoryBagTabEquip},
		{name: "etc", item: session.InventoryItem{Type: 3}, tab: inventoryBagTabEtc},
		{name: "card", item: session.InventoryItem{Type: 6}, tab: inventoryBagTabEtc},
		{name: "card with stale equipment flag", item: session.InventoryItem{Type: 6, Equip: true}, tab: inventoryBagTabEtc},
		{name: "ammo", item: session.InventoryItem{Type: 10}, tab: inventoryBagTabEtc},
		{name: "ammo with equipment flag", item: session.InventoryItem{Type: 10, Equip: true}, tab: inventoryBagTabEtc},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inventoryItemTab(tc.item); got != tc.tab {
				t.Fatalf("tab = %d, want %d", got, tc.tab)
			}
		})
	}
}

func TestInventoryBagSeparatesEquipCapabilityFromEquipTab(t *testing.T) {
	if inventoryItemCanEquip(session.InventoryItem{Type: db.ItemTypeCard, Equip: true}) {
		t.Fatal("card should not be equip-capable")
	}
	if !inventoryItemCanEquip(session.InventoryItem{Type: db.ItemTypeAmmo}) {
		t.Fatal("ammo should stay equip-capable")
	}
}

func TestInventoryBagClampScrollUsesPixelOffset(t *testing.T) {
	items := make([]session.InventoryItem, inventoryBagCols*(inventoryBagRows+2))
	for i := range items {
		items[i] = session.InventoryItem{Index: uint16(i + 1), ItemID: 501, Type: db.ItemTypeHealing}
	}
	sessionState := &session.Session{Inventory: session.Inventory{Items: items}}
	window := InventoryBagWindow{tab: inventoryBagTabItem}
	scroll := window.ensureScrollSignal()

	scroll.Set(999)
	window.ClampScroll(sessionState)
	if got, want := scroll.Get(), float32(2*inventoryBagCell); got != want {
		t.Fatalf("clamped scroll = %.1f, want %.1f", got, want)
	}

	scroll.Set(-8)
	window.ClampScroll(sessionState)
	if got := scroll.Get(); got != 0 {
		t.Fatalf("negative scroll clamped to %.1f, want 0", got)
	}
}

func TestInventoryBagWindowHeightEndsAtGridBottom(t *testing.T) {
	if got, want := inventoryBagHeight-ROWindowTitleHeight, inventoryBagViewH; got != want {
		t.Fatalf("inventory content height = %d, want grid height %d", got, want)
	}
}

func TestInventoryBagMatchesCharacterWindowWidth(t *testing.T) {
	if inventoryBagWidth != characterWindowWidth {
		t.Fatalf("inventory width = %d, want character width %d", inventoryBagWidth, characterWindowWidth)
	}
	wantViewWidth := inventoryBagWidth - inventoryBagTabRail - verticalTabDividerW
	if inventoryBagViewW != wantViewWidth {
		t.Fatalf("inventory view width = %d, want remaining width %d", inventoryBagViewW, wantViewWidth)
	}
	if inventoryBagGridW+ROScrollbarGutter >= inventoryBagViewW {
		t.Fatal("wider inventory should leave trailing room while keeping the grid left aligned")
	}
}

func TestInventoryGridStartsAtLeftEdge(t *testing.T) {
	grid := newInventoryGridWidget(inventoryGridConfig{
		items:     []session.InventoryItem{{Index: 1, ItemID: 501}},
		viewWidth: inventoryBagViewW,
	})
	grid.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(inventoryBagViewW, inventoryBagViewH)))

	if got := grid.indexAt(geometry.Pt(inventoryBagCell/2, inventoryBagCell/2)); got != 0 {
		t.Fatalf("first cell hit index %d, want 0", got)
	}
	if got := grid.cellBounds(0).Min.X; got != 0 {
		t.Fatalf("first cell x = %.1f, want left edge 0", got)
	}
}

func TestInventoryGridDrawsOnlyBottomShadowInEveryCell(t *testing.T) {
	grid := newInventoryGridWidget(inventoryGridConfig{
		items:     []session.InventoryItem{{Index: 1, ItemID: 501}},
		viewWidth: inventoryBagViewW,
	})
	grid.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(inventoryBagViewW, inventoryBagViewH)))
	canvas := &uitest.MockCanvas{}
	grid.Draw(widget.NewContext(), canvas)

	wantCells := inventoryBagCols * inventoryBagRows
	if len(canvas.RoundRects) != wantCells {
		t.Fatalf("cell shadows = %d, want one for each of %d cells", len(canvas.RoundRects), wantCells)
	}
	if len(canvas.Rects) != 1 || canvas.Rects[0].Bounds != grid.Bounds() {
		t.Fatalf("grid rectangles = %v, want only the window-body background", canvas.Rects)
	}
	if len(canvas.StrokeRects) != 0 {
		t.Fatalf("grid lines = %d, want none", len(canvas.StrokeRects))
	}
	first := canvas.RoundRects[0]
	cell := grid.cellBounds(0)
	height := (cell.Height() - inventoryBagCellShadowInset*2) / 2
	wantBounds := geometry.NewRect(
		cell.Min.X+inventoryBagCellShadowInset,
		cell.Max.Y-inventoryBagCellShadowInset-height,
		cell.Width()-inventoryBagCellShadowInset*2,
		height,
	)
	if first.Bounds != wantBounds || first.Radius != inventoryBagCellShadowRadius {
		t.Fatalf("first cell shadow = bounds %v radius %.1f, want %v radius %.1f", first.Bounds, first.Radius, wantBounds, inventoryBagCellShadowRadius)
	}
	wantColor := rotheme.Default.Colors.ButtonHover
	wantColor.A = inventoryBagCellShadowAlpha
	uitest.AssertColorEqual(t, first.Color, wantColor)
}

func TestPushcartGridUsesBottomCellShadows(t *testing.T) {
	grid := newInventoryGridWidget(inventoryGridConfig{
		items:     []session.InventoryItem{{Index: 1, ItemID: 501}},
		cols:      cartGridCols,
		minRows:   cartGridRows,
		cellSize:  cartGridCell,
		viewWidth: cartGridViewW,
	})
	grid.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(cartGridViewW, cartGridViewH)))
	canvas := &uitest.MockCanvas{}
	grid.Draw(widget.NewContext(), canvas)

	if got, want := len(canvas.RoundRects), cartGridCols*cartGridRows; got != want {
		t.Fatalf("pushcart cell shadows = %d, want %d", got, want)
	}
	if len(canvas.StrokeRects) != 0 {
		t.Fatalf("pushcart grid lines = %d, want none", len(canvas.StrokeRects))
	}
}

func TestInventoryGridUsesScrollableContentCoordinates(t *testing.T) {
	items := make([]session.InventoryItem, inventoryBagCols*inventoryBagRows+1)
	for i := range items {
		items[i] = session.InventoryItem{Index: uint16(i + 1), ItemID: 501, Type: db.ItemTypeHealing}
	}
	grid := newInventoryGridWidget(inventoryGridConfig{items: items})
	size := grid.Layout(widget.NewContext(), geometry.Constraints{
		MinWidth:  inventoryBagViewW,
		MaxWidth:  inventoryBagViewW,
		MinHeight: 0,
		MaxHeight: geometry.Infinity,
	})

	if got, want := size.Width, float32(inventoryBagViewW); got != want {
		t.Fatalf("grid width = %.1f, want %.1f", got, want)
	}
	if got, want := size.Height, float32((inventoryBagRows+1)*inventoryBagCell); got != want {
		t.Fatalf("grid height = %.1f, want %.1f", got, want)
	}
	if got, want := grid.indexAt(geometry.Pt(inventoryBagCell/2, inventoryBagRows*inventoryBagCell+inventoryBagCell/2)), inventoryBagCols*inventoryBagRows; got != want {
		t.Fatalf("scrolled content index = %d, want %d", got, want)
	}
	if got := grid.indexAt(geometry.Pt(inventoryBagGridW+1, inventoryBagCell/2)); got != -1 {
		t.Fatalf("scrollbar gutter hit index = %d, want -1", got)
	}
}

func TestInventoryItemDisplayNameAddsSlotCountForIdentifiedItems(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "idnum2itemdisplaynametable.txt"), []byte("2607#Clip#\n2608#Ring [1]#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "num2itemdisplaynametable.txt"), []byte("2607#Accessory#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "itemslotcounttable.txt"), []byte("2607#1#\n2608#1#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "cardprefixnametable.txt"), []byte("4001#Poring#\n4002#Fabre#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "cardpostfixnametable.txt"), []byte("4002#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}

	if got := inventoryItemDisplayName(manager, session.InventoryItem{ItemID: 2607, Identified: true}); got != "Clip [1]" {
		t.Fatalf("identified slotted name = %q, want Clip [1]", got)
	}
	if got := inventoryItemDisplayName(manager, session.InventoryItem{ItemID: 2607, Identified: false}); got != "Accessory" {
		t.Fatalf("unidentified slotted name = %q, want Accessory", got)
	}
	if got := inventoryItemDisplayName(manager, session.InventoryItem{ItemID: 2608, Identified: true}); got != "Ring [1]" {
		t.Fatalf("pre-suffixed slotted name = %q, want Ring [1]", got)
	}
	if got := inventoryItemDisplayName(manager, session.InventoryItem{ItemID: 2607, Type: db.ItemTypeArmor, Identified: true, Cards: [4]uint16{4001, 4001, 4002}}); got != "Double Poring Clip Fabre [1]" {
		t.Fatalf("carded name = %q, want Double Poring Clip Fabre [1]", got)
	}
}

func TestStorageAcceptInventoryDropWithoutNetworkConsumesDrop(t *testing.T) {
	window := StorageWindow{}
	sessionState := &session.Session{Storage: session.Storage{Open: true}}
	ctx := Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	window.OpenWindow(ctx)
	ok := window.AcceptInventoryDrop(Context{Session: sessionState}, session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, window.Window.x+12, window.Window.y+20)
	if !ok {
		t.Fatal("drop over storage was not consumed")
	}
}

func TestCartAcceptInventoryDropWithoutNetworkConsumesDrop(t *testing.T) {
	window := CartWindow{}
	sessionState := &session.Session{Cart: session.Cart{Open: true}}
	ctx := Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	window.OpenWindow(ctx)
	ok := window.AcceptInventoryDrop(Context{Session: sessionState}, session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, window.Window.x+12, window.Window.y+20)
	if !ok {
		t.Fatal("drop over cart was not consumed")
	}
}

func TestStorageAndCartAcceptCrossDropsWithoutNetwork(t *testing.T) {
	sessionState := &session.Session{
		Storage: session.Storage{Open: true},
		Cart:    session.Cart{Open: true},
	}
	ctx := Context{Session: sessionState, ScreenW: 800, ScreenH: 600}

	storage := StorageWindow{}
	storage.OpenWindow(ctx)
	if !storage.AcceptCartDrop(ctx, session.InventoryItem{Index: 4, ItemID: 938, Amount: 2}, storage.Window.x+12, storage.Window.y+20) {
		t.Fatal("drop from cart over storage was not consumed")
	}

	cart := CartWindow{}
	cart.OpenWindow(ctx)
	if !cart.AcceptStorageDrop(ctx, session.InventoryItem{Index: 5, ItemID: 938, Amount: 2}, cart.Window.x+12, cart.Window.y+20) {
		t.Fatal("drop from storage over cart was not consumed")
	}
}

func TestStorageItemAtUsesTableViewRowsAtTop(t *testing.T) {
	sessionState := &session.Session{
		Storage: session.Storage{
			Open:  true,
			Items: []session.InventoryItem{{Index: 4, ItemID: 938, Amount: 2}},
		},
	}
	window := StorageWindow{}
	window.EnsureWindow(storageWindowWidth, storageWindowHeight)
	window.Window.OpenAt(80, 90, nil)

	item, row, ok := window.itemAt(sessionState, window.x+storageTabRailW+verticalTabDividerW+8, window.y+storageWindowTitleH+1)

	if !ok {
		t.Fatal("storage row at top of table was not found")
	}
	if row != 0 || item.Index != 4 {
		t.Fatalf("itemAt = row %d item %+v, want row 0 index 4", row, item)
	}
}

func TestStorageWindowUsesROBrowserCategories(t *testing.T) {
	tests := []struct {
		name     string
		itemType uint8
		category storageCategory
	}{
		{name: "healing", itemType: db.ItemTypeHealing, category: storageCategoryItem},
		{name: "usable", itemType: db.ItemTypeUsable, category: storageCategoryItem},
		{name: "delayed consumable", itemType: db.ItemTypeDelayConsume, category: storageCategoryItem},
		{name: "cash", itemType: db.ItemTypeCash, category: storageCategoryKafra},
		{name: "armor", itemType: db.ItemTypeArmor, category: storageCategoryArmor},
		{name: "shadow gear", itemType: db.ItemTypeShadowGear, category: storageCategoryArmor},
		{name: "pet egg", itemType: db.ItemTypePetEgg, category: storageCategoryArmor},
		{name: "weapon", itemType: db.ItemTypeWeapon, category: storageCategoryArms},
		{name: "pet armor", itemType: db.ItemTypePetArmor, category: storageCategoryArms},
		{name: "ammo", itemType: db.ItemTypeAmmo, category: storageCategoryAmmo},
		{name: "card", itemType: db.ItemTypeCard, category: storageCategoryCard},
		{name: "etc", itemType: db.ItemTypeEtc, category: storageCategoryEtc},
		{name: "unknown", itemType: db.ItemTypeUnknown, category: storageCategoryEtc},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := storageItemCategory(session.InventoryItem{Type: tc.itemType}); got != tc.category {
				t.Fatalf("category = %d, want %d", got, tc.category)
			}
		})
	}

	sessionState := &session.Session{Storage: session.Storage{Items: []session.InventoryItem{
		{Index: 2, ItemID: 1201, Type: db.ItemTypeWeapon},
		{Index: 1, ItemID: 1101, Type: db.ItemTypeWeapon},
		{Index: 3, ItemID: 501, Type: db.ItemTypeHealing},
	}}}
	window := StorageWindow{tab: storageCategoryArms}
	items := window.tabItems(sessionState)
	if len(items) != 2 || items[0].Index != 1 || items[1].Index != 2 {
		t.Fatalf("storage arms tab = %+v, want sorted indexes 1 and 2", items)
	}
}

func TestStorageWindowSelectsFirstNonEmptyCategory(t *testing.T) {
	sessionState := &session.Session{Storage: session.Storage{Items: []session.InventoryItem{
		{Index: 2, ItemID: 1201, Type: db.ItemTypeWeapon},
	}}}
	window := StorageWindow{tab: storageCategoryItem}

	window.selectFirstNonEmptyTab(sessionState)

	if window.tab != storageCategoryArms {
		t.Fatalf("storage tab = %d, want arms tab", window.tab)
	}
}

func TestStorageWindowTabRailDoesNotHitItems(t *testing.T) {
	sessionState := &session.Session{Storage: session.Storage{Items: []session.InventoryItem{
		{Index: 1, ItemID: 501, Type: db.ItemTypeHealing},
	}}}
	window := StorageWindow{tab: storageCategoryItem}
	window.EnsureWindow(storageWindowWidth, storageWindowHeight)
	window.Window.OpenAt(80, 90, nil)

	if _, _, ok := window.itemAt(sessionState, window.x+storageTabRailW/2, window.y+storageWindowTitleH+1); ok {
		t.Fatal("storage tab rail hit an item row")
	}
}

func TestInventoryBagShowsCartButtonOnlyWhenPlayerHasCart(t *testing.T) {
	if inventoryBagHasCart(Context{}) {
		t.Fatal("empty context should not show cart button")
	}
	world := worldstate.New()
	world.Player.HasCartState = true
	world.Player.HasCart = true
	if !inventoryBagHasCart(Context{World: world}) {
		t.Fatal("player cart state should show cart button")
	}
	if !inventoryBagHasCart(Context{Session: &session.Session{Selected: session.Character{ID: 150004, Option: inventoryBagCartOptionMask}}}) {
		t.Fatal("selected character cart option should show cart button")
	}
	world.Player.HasCart = false
	if inventoryBagHasCart(Context{World: world, Session: &session.Session{Selected: session.Character{ID: 150004, Option: inventoryBagCartOptionMask}}}) {
		t.Fatal("explicit world cart removal should hide cart button")
	}
}

func TestInventoryBagTooltipTracksHoveredItem(t *testing.T) {
	window := InventoryBagWindow{}
	item := session.InventoryItem{Index: 8, ItemID: 501, Identified: true}
	ctx := Context{Input: input.NewState(), ScreenW: 800, ScreenH: 600, UIManager: NewManager()}

	window.showTooltip(ctx, item)
	if !window.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	if got := window.tooltip.Text(); got != "item 501" {
		t.Fatalf("tooltip text = %q", got)
	}

	window.hideTooltip()
	if window.tooltip.Open() {
		t.Fatal("tooltip not cleared")
	}
}

func TestStorageDragReleaseOverInventoryWithdraws(t *testing.T) {
	inputState := input.NewState()
	inputState.SetMousePosition(40, 40)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)

	inventory := InventoryBagWindow{}
	inventory.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	inventory.Window.OpenAt(24, 24, nil)
	storage := StorageWindow{
		dragItem:   session.InventoryItem{Index: 9, ItemID: 938, Amount: 2},
		dragActive: true,
		dragFrom:   time.Now().Add(-time.Second),
	}
	consumed := storage.UpdateDrag(Context{Input: inputState}, &inventory, nil)
	if !consumed {
		t.Fatal("storage drag release was not consumed")
	}
	if storage.dragActive {
		t.Fatal("storage drag stayed active after release")
	}
}

func TestInventoryDropAmountClampsToAvailableAmount(t *testing.T) {
	stack := session.InventoryItem{Type: db.ItemTypeEtc, Amount: 9}
	if got := inventoryDropAmount(stack, 5); got != 5 {
		t.Fatalf("stack drop amount = %d, want 5", got)
	}
	if got := inventoryDropAmount(stack, 20); got != 9 {
		t.Fatalf("oversized stack drop amount = %d, want 9", got)
	}
	if got := inventoryDropAmount(stack, 0); got != 1 {
		t.Fatalf("zero stack drop amount = %d, want 1", got)
	}
}

func TestInventoryDropAmountUsesOneForNonStackableItems(t *testing.T) {
	item := session.InventoryItem{Type: db.ItemTypeWeapon, Amount: 9}
	if got := inventoryDropAmount(item, 9); got != 1 {
		t.Fatalf("non-stackable drop amount = %d, want 1", got)
	}
}

func TestInventoryDropMaxAmountFitsPacketRange(t *testing.T) {
	item := session.InventoryItem{Type: db.ItemTypeEtc, Amount: int(^uint16(0)) + 20}
	if got := inventoryDropMaxAmount(item); got != ^uint16(0) {
		t.Fatalf("drop max = %d, want %d", got, ^uint16(0))
	}
}

func TestAmountPromptParsingClampsToAvailableAndPacketRange(t *testing.T) {
	if got, ok := parseAmount("99999", 9); !ok || got != 9 {
		t.Fatalf("available clamp = %d, %t, want 9, true", got, ok)
	}
	if got, ok := parseAmount("99999", ^uint16(0)); !ok || got != ^uint16(0) {
		t.Fatalf("packet clamp = %d, %t, want %d, true", got, ok, ^uint16(0))
	}
	if _, ok := parseAmount("", 9); ok {
		t.Fatal("blank amount should not submit")
	}
}

func TestInventoryStackDropOpensPromptDefaultedToAvailableAmount(t *testing.T) {
	window := InventoryBagWindow{}
	window.requestDrop(Context{}, session.InventoryItem{Index: 7, ItemID: 938, Type: db.ItemTypeEtc, Amount: 9})
	if !window.amountPrompt.IsOpen() {
		t.Fatal("stack drop did not open an amount prompt")
	}
	if window.amountPrompt.value != "9" || window.amountPrompt.max != 9 {
		t.Fatalf("amount prompt value=%q max=%d, want 9 and 9", window.amountPrompt.value, window.amountPrompt.max)
	}
}

func TestInventoryDropAmountPromptEscapeCancelsWithoutDropping(t *testing.T) {
	inputState := input.NewState()
	window := InventoryBagWindow{}
	dropped := false
	ctx := Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	window.amountPrompt.Open(ctx, "Enter amount to drop", 9, 9, func(uint16) {
		dropped = true
	})

	inputState.SetKey(input.KeyEscape, true)
	if !window.UpdateDropPrompt(ctx) {
		t.Fatal("open drop prompt did not consume Escape")
	}
	if window.amountPrompt.IsOpen() {
		t.Fatal("drop prompt remained open after Escape")
	}
	if dropped {
		t.Fatal("Escape submitted the item drop")
	}
}

func TestInventoryNonStackableDropSkipsPrompt(t *testing.T) {
	window := InventoryBagWindow{}
	window.requestDrop(Context{}, session.InventoryItem{Index: 7, ItemID: 1201, Type: db.ItemTypeWeapon, Amount: 9})
	if window.amountPrompt.IsOpen() {
		t.Fatal("non-stackable drop opened an amount prompt")
	}
}

func TestIdentifyWindowShowsOnlyUnidentifiedEquipmentFromServerList(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 512, Type: 0, Identified: false},
				{Index: 5, ItemID: 1201, Type: 5, Identified: false, Equip: true},
				{Index: 7, ItemID: 1202, Type: 5, Identified: true, Equip: true},
			},
		},
	}
	window := IdentifyWindow{}
	window.OpenList(Context{Session: sessionState, ScreenW: 800, ScreenH: 600}, network.ItemIdentifyList{Indexes: []uint16{3, 5, 7, 9}})

	items := window.items(sessionState)
	if len(items) != 1 || items[0].Index != 5 {
		t.Fatalf("identify items = %+v, want only index 5", items)
	}
	if !window.IsOpen() {
		t.Fatal("identify window did not open")
	}
}

func TestMakingItemWindowOpensServerList(t *testing.T) {
	window := MakingItemWindow{}
	window.OpenList(Context{ScreenW: 800, ScreenH: 600}, network.MakingItemList{
		Items: []network.MakingItemOption{
			{ItemID: 501, Material: [3]uint16{1000, 990, 5}},
			{ItemID: 713},
		},
	})

	if !window.IsOpen() {
		t.Fatal("making item window did not open")
	}
	if len(window.items) != 2 || window.items[0].ItemID != 501 || window.items[0].Material != [3]uint16{1000, 990, 5} {
		t.Fatalf("making item window items = %+v", window.items)
	}

	window.OpenList(Context{ScreenW: 800, ScreenH: 600}, network.MakingItemList{})
	if window.IsOpen() {
		t.Fatal("making item window stayed open for empty list")
	}
}
