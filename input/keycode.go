package input

import "github.com/gogpu/gpucontext"

// KeyCodeFromName resolves a layout-independent physical key name. Names use
// the same vocabulary as browser KeyboardEvent.code so scripts remain clear
// across QWERTY, AZERTY, and other layouts.
func KeyCodeFromName(name string) (KeyCode, bool) {
	code, ok := keyCodesByName[name]
	return code, ok
}

// KeyCodeName is the layout-independent name used by keyboard callbacks.
func KeyCodeName(code KeyCode) string {
	if name, ok := keyNamesByCode[code]; ok {
		return name
	}
	return "Unknown"
}

var keyNamesByCode = func() map[KeyCode]string {
	names := make(map[KeyCode]string, len(keyCodesByName))
	for name, code := range keyCodesByName {
		names[code] = name
	}
	return names
}()

var keyCodesByName = map[string]KeyCode{
	"KeyA": gpucontext.KeyA, "KeyB": gpucontext.KeyB, "KeyC": gpucontext.KeyC,
	"KeyD": gpucontext.KeyD, "KeyE": gpucontext.KeyE, "KeyF": gpucontext.KeyF,
	"KeyG": gpucontext.KeyG, "KeyH": gpucontext.KeyH, "KeyI": gpucontext.KeyI,
	"KeyJ": gpucontext.KeyJ, "KeyK": gpucontext.KeyK, "KeyL": gpucontext.KeyL,
	"KeyM": gpucontext.KeyM, "KeyN": gpucontext.KeyN, "KeyO": gpucontext.KeyO,
	"KeyP": gpucontext.KeyP, "KeyQ": gpucontext.KeyQ, "KeyR": gpucontext.KeyR,
	"KeyS": gpucontext.KeyS, "KeyT": gpucontext.KeyT, "KeyU": gpucontext.KeyU,
	"KeyV": gpucontext.KeyV, "KeyW": gpucontext.KeyW, "KeyX": gpucontext.KeyX,
	"KeyY": gpucontext.KeyY, "KeyZ": gpucontext.KeyZ,

	"Digit0": gpucontext.Key0, "Digit1": gpucontext.Key1, "Digit2": gpucontext.Key2,
	"Digit3": gpucontext.Key3, "Digit4": gpucontext.Key4, "Digit5": gpucontext.Key5,
	"Digit6": gpucontext.Key6, "Digit7": gpucontext.Key7, "Digit8": gpucontext.Key8,
	"Digit9": gpucontext.Key9,

	"F1": gpucontext.KeyF1, "F2": gpucontext.KeyF2, "F3": gpucontext.KeyF3,
	"F4": gpucontext.KeyF4, "F5": gpucontext.KeyF5, "F6": gpucontext.KeyF6,
	"F7": gpucontext.KeyF7, "F8": gpucontext.KeyF8, "F9": gpucontext.KeyF9,
	"F10": gpucontext.KeyF10, "F11": gpucontext.KeyF11, "F12": gpucontext.KeyF12,

	"Escape": gpucontext.KeyEscape, "Tab": gpucontext.KeyTab,
	"Backspace": gpucontext.KeyBackspace, "Enter": gpucontext.KeyEnter,
	"Space": gpucontext.KeySpace, "Insert": gpucontext.KeyInsert,
	"Delete": gpucontext.KeyDelete, "Home": gpucontext.KeyHome, "End": gpucontext.KeyEnd,
	"PageUp": gpucontext.KeyPageUp, "PageDown": gpucontext.KeyPageDown,
	"ArrowLeft": gpucontext.KeyLeft, "ArrowRight": gpucontext.KeyRight,
	"ArrowUp": gpucontext.KeyUp, "ArrowDown": gpucontext.KeyDown,

	"ShiftLeft": gpucontext.KeyLeftShift, "ShiftRight": gpucontext.KeyRightShift,
	"ControlLeft": gpucontext.KeyLeftControl, "ControlRight": gpucontext.KeyRightControl,
	"AltLeft": gpucontext.KeyLeftAlt, "AltRight": gpucontext.KeyRightAlt,
	"MetaLeft": gpucontext.KeyLeftSuper, "MetaRight": gpucontext.KeyRightSuper,

	"Minus": gpucontext.KeyMinus, "Equal": gpucontext.KeyEqual,
	"BracketLeft": gpucontext.KeyLeftBracket, "BracketRight": gpucontext.KeyRightBracket,
	"Backslash": gpucontext.KeyBackslash, "Semicolon": gpucontext.KeySemicolon,
	"Quote": gpucontext.KeyApostrophe, "Backquote": gpucontext.KeyGrave,
	"Comma": gpucontext.KeyComma, "Period": gpucontext.KeyPeriod, "Slash": gpucontext.KeySlash,

	"Numpad0": gpucontext.KeyNumpad0, "Numpad1": gpucontext.KeyNumpad1,
	"Numpad2": gpucontext.KeyNumpad2, "Numpad3": gpucontext.KeyNumpad3,
	"Numpad4": gpucontext.KeyNumpad4, "Numpad5": gpucontext.KeyNumpad5,
	"Numpad6": gpucontext.KeyNumpad6, "Numpad7": gpucontext.KeyNumpad7,
	"Numpad8": gpucontext.KeyNumpad8, "Numpad9": gpucontext.KeyNumpad9,
	"NumpadDecimal": gpucontext.KeyNumpadDecimal, "NumpadDivide": gpucontext.KeyNumpadDivide,
	"NumpadMultiply": gpucontext.KeyNumpadMultiply, "NumpadSubtract": gpucontext.KeyNumpadSubtract,
	"NumpadAdd": gpucontext.KeyNumpadAdd, "NumpadEnter": gpucontext.KeyNumpadEnter,

	"CapsLock": gpucontext.KeyCapsLock, "ScrollLock": gpucontext.KeyScrollLock,
	"NumLock": gpucontext.KeyNumLock, "PrintScreen": gpucontext.KeyPrintScreen,
	"Pause": gpucontext.KeyPause,
}

func legacyKeyForCode(code KeyCode) (Key, bool) {
	switch code {
	case gpucontext.KeyEnter:
		return KeyEnter, true
	case gpucontext.KeyEscape:
		return KeyEscape, true
	case gpucontext.KeyTab:
		return KeyTab, true
	case gpucontext.KeyUp:
		return KeyArrowUp, true
	case gpucontext.KeyDown:
		return KeyArrowDown, true
	case gpucontext.KeyLeft:
		return KeyArrowLeft, true
	case gpucontext.KeyRight:
		return KeyArrowRight, true
	case gpucontext.KeyBackspace:
		return KeyBackspace, true
	case gpucontext.KeyLeftShift, gpucontext.KeyRightShift:
		return KeyShift, true
	case gpucontext.KeyLeftControl, gpucontext.KeyRightControl:
		return KeyCtrl, true
	case gpucontext.KeyLeftAlt, gpucontext.KeyRightAlt:
		return KeyAlt, true
	case gpucontext.KeyG:
		return KeyG, true
	case gpucontext.KeyL:
		return KeyL, true
	case gpucontext.Key1:
		return Key1, true
	case gpucontext.Key2:
		return Key2, true
	case gpucontext.Key3:
		return Key3, true
	case gpucontext.Key4:
		return Key4, true
	case gpucontext.Key5:
		return Key5, true
	case gpucontext.Key6:
		return Key6, true
	case gpucontext.Key7:
		return Key7, true
	case gpucontext.Key8:
		return Key8, true
	case gpucontext.Key9:
		return Key9, true
	case gpucontext.KeyQ:
		return KeyQ, true
	case gpucontext.KeyW:
		return KeyW, true
	case gpucontext.KeyE:
		return KeyE, true
	case gpucontext.KeyR:
		return KeyR, true
	case gpucontext.KeyT:
		return KeyT, true
	case gpucontext.KeyY:
		return KeyY, true
	case gpucontext.KeyU:
		return KeyU, true
	case gpucontext.KeyI:
		return KeyI, true
	case gpucontext.KeyO:
		return KeyO, true
	case gpucontext.KeyF1:
		return KeyF1, true
	case gpucontext.KeyF2:
		return KeyF2, true
	case gpucontext.KeyF3:
		return KeyF3, true
	case gpucontext.KeyF4:
		return KeyF4, true
	case gpucontext.KeyF5:
		return KeyF5, true
	case gpucontext.KeyF6:
		return KeyF6, true
	case gpucontext.KeyF7:
		return KeyF7, true
	case gpucontext.KeyF8:
		return KeyF8, true
	case gpucontext.KeyF9:
		return KeyF9, true
	case gpucontext.KeyF12:
		return KeyF12, true
	default:
		return 0, false
	}
}
