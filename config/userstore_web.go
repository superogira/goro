//go:build js && wasm

package config

import (
	"errors"
	"strings"
	"syscall/js"
)

var errBrowserStorageUnavailable = errors.New("browser storage unavailable")

// userSettingsStoreKey holds the browser-local copy of the user's settings
// (the same ini text SaveUserSettings writes to goro.ini on native builds).
// localStorage survives page reloads, which the wasm filesystem does not.
const userSettingsStoreKey = "goroUserSettings"

func localStorageGet(key string) string {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return ""
	}
	raw := storage.Call("getItem", key)
	if raw.Type() != js.TypeString {
		return ""
	}
	return raw.String()
}

func localStorageSet(key, value string) bool {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return false
	}
	storage.Call("setItem", key, value)
	return true
}

// applyStoredUserINI applies the settings the user saved from the settings
// window. Stored values win over the deployment goro.ini fetched by
// applyServerINI but lose to the URL parameters applied afterwards.
func applyStoredUserINI(cfg *Config) {
	raw := localStorageGet(userSettingsStoreKey)
	if strings.TrimSpace(raw) == "" {
		return
	}
	if err := applyINI(cfg, strings.NewReader(raw)); err != nil {
		return
	}
}

// writeUserSettings stores the settings ini in localStorage on web builds.
func writeUserSettings(values map[string]map[string]string) (string, error) {
	data := upsertINIValues(localStorageGet(userSettingsStoreKey), values)
	if !localStorageSet(userSettingsStoreKey, data) {
		return "", errBrowserStorageUnavailable
	}
	return "browser settings", nil
}
