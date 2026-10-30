package maps

func InfoNotifyMessage(result uint16) string {
	switch result {
	case 0:
		return "Unable to teleport in this area."
	case 1:
		return "Saved point cannot be memorized."
	case 2:
		return "This skill cannot be used in this area."
	case 3:
		return "This item cannot be used in this area."
	default:
		return "Action cannot be used in this area."
	}
}
