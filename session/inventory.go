package session

func RemoveInventoryItem(s *Session, index uint16, amount int) {
	if s == nil || index == 0 {
		return
	}
	if amount <= 0 {
		amount = 1
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].Index != index {
			continue
		}
		s.Inventory.Items[i].Amount -= amount
		if s.Inventory.Items[i].Amount > 0 {
			return
		}
		s.Inventory.Items = append(s.Inventory.Items[:i], s.Inventory.Items[i+1:]...)
		return
	}
}

func FindInventoryItem(s *Session, index uint16) (InventoryItem, bool) {
	if s == nil || index == 0 {
		return InventoryItem{}, false
	}
	for _, item := range s.Inventory.Items {
		if item.Index == index {
			return item, true
		}
	}
	return InventoryItem{}, false
}

func CloseStorage(s *Session) {
	if s == nil {
		return
	}
	s.Storage.Open = false
	s.Storage.PasswordPending = false
	s.Storage.Items = nil
	s.Storage.Amount = 0
	s.Storage.MaxAmount = 0
}
