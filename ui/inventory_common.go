package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

const inventoryIconSize = 24

var (
	inventoryTextColor = TextColor
)

func sortedInventoryItems(s *session.Session) []session.InventoryItem {
	if s == nil {
		return nil
	}
	items := make([]session.InventoryItem, 0, len(s.Inventory.Items))
	items = append(items, s.Inventory.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Index < items[j].Index
	})
	return items
}

func inventoryItemDisplayName(manager *res.Manager, item session.InventoryItem) string {
	if manager != nil {
		if name, ok := manager.ItemDisplayName(int(item.ItemID), item.Identified); ok && strings.TrimSpace(name) != "" {
			return inventoryItemDisplayNameWithCardsAndSlots(manager, item, name)
		}
	}
	return fmt.Sprintf("item %d", item.ItemID)
}

func inventoryItemDisplayNameWithCardsAndSlots(manager *res.Manager, item session.InventoryItem, name string) string {
	if !item.Identified || manager == nil {
		return name
	}
	hasSlotSuffix := itemDisplayNameHasSlotSuffix(name)
	prefix, postfix := inventoryItemCardNameParts(manager, item)
	if prefix != "" {
		name = prefix + name
	}
	if postfix != "" {
		name += postfix
	}
	if !hasSlotSuffix {
		if slotCount, ok := manager.ItemSlotCount(int(item.ItemID)); ok && slotCount > 0 && inventoryItemCanShowSlots(item) {
			name = fmt.Sprintf("%s [%d]", name, slotCount)
		}
	}
	return name
}

func inventoryItemCardNameParts(manager *res.Manager, item session.InventoryItem) (string, string) {
	if manager == nil || !inventoryItemCanShowCards(item) {
		return "", ""
	}
	type cardInfo struct {
		count   int
		postfix bool
		prefix  string
	}
	order := make([]uint16, 0, len(item.Cards))
	cards := make(map[uint16]*cardInfo)
	for _, cardID := range item.Cards {
		if cardID == 0 || cardID == 0x00ff || cardID == 0x00fe || cardID == 0xff00 {
			continue
		}
		info := cards[cardID]
		if info == nil {
			prefixName, ok := manager.ItemCardPrefixName(int(cardID))
			if !ok || strings.TrimSpace(prefixName) == "" {
				continue
			}
			info = &cardInfo{prefix: prefixName, postfix: manager.ItemCardPostfix(int(cardID))}
			cards[cardID] = info
			order = append(order, cardID)
		}
		info.count++
	}
	duplicatePrefix := []string{"", "Double ", "Triple ", "Quadruple "}
	var prefix, postfix strings.Builder
	for _, cardID := range order {
		info := cards[cardID]
		if info == nil || info.count <= 0 {
			continue
		}
		countPrefix := ""
		if info.count < len(duplicatePrefix) {
			countPrefix = duplicatePrefix[info.count-1]
		}
		part := countPrefix + info.prefix
		if info.postfix {
			postfix.WriteString(" ")
			postfix.WriteString(part)
		} else {
			prefix.WriteString(part)
			prefix.WriteString(" ")
		}
	}
	return prefix.String(), postfix.String()
}

func inventoryItemCanShowCards(item session.InventoryItem) bool {
	switch item.Type {
	case db.ItemTypeWeapon, db.ItemTypeArmor, db.ItemTypeShadowGear:
		return true
	default:
		return false
	}
}

func inventoryItemCanShowSlots(item session.InventoryItem) bool {
	return item.Cards[0] != 0x00ff && item.Cards[0] != 0x00fe && item.Cards[0] != 0xff00 && item.Cards[3] != 0x0001
}

func itemDisplayNameHasSlotSuffix(name string) bool {
	name = strings.TrimSpace(name)
	if !strings.HasSuffix(name, "]") {
		return false
	}
	open := strings.LastIndex(name, "[")
	if open < 0 || open+1 >= len(name)-1 {
		return false
	}
	_, err := strconv.Atoi(strings.TrimSpace(name[open+1 : len(name)-1]))
	return err == nil
}
