//go:build !js || !wasm

package ui

import (
	"github.com/kivutar/goro/session"
)

// Native stubs: no DOM panels exist, so the callers fall back to the
// canvas windows.

func confirmWebSync(title, message string, okOnly, open bool) bool {
	return false
}

type friendsWebState struct {
	open      bool
	tab       string
	title     string
	friends   []session.Friend
	party     session.Party
	canManage bool
	selfAID   uint32
}

func friendsWebSync(state friendsWebState) bool { return false }

type friendSetupWebState struct {
	open         bool
	openStranger bool
	openFriends  bool
	alert        bool
}

func friendSetupWebSync(state friendSetupWebState) bool { return false }

type partySetupWebState struct {
	open         bool
	expShare     uint32
	refuseInvite bool
}

func partySetupWebSync(state partySetupWebState) bool { return false }

type whisperWebWindow struct {
	target string
	lines  []whisperWebLine
}

type whisperWebLine struct {
	text string
	kind string
}

func whisperWebEnabled() bool { return false }

func whisperWebSync(windows []whisperWebWindow) bool { return false }

func drainSocialWebActions(prefix string) []string { return nil }
