package game

import (
	"github.com/kivutar/goro/db"
)

const inventoryIconSize = 24

func inventoryItemTypeIsEquipment(itemType uint8) bool {
	switch itemType {
	case db.ItemTypeArmor, db.ItemTypeWeapon, db.ItemTypePetEgg, db.ItemTypePetArmor, db.ItemTypeAmmo, db.ItemTypeShadowGear:
		return true
	default:
		return false
	}
}

func inventoryItemTypeIsAmmo(itemType uint8) bool {
	return itemType == db.ItemTypeAmmo
}

func inventoryItemDefaultEquipLocation(itemType uint8) uint16 {
	if inventoryItemTypeIsAmmo(itemType) {
		return db.EquipAmmo
	}
	return 0
}
