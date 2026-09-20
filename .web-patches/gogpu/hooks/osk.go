// Package hooks provides shared callbacks between the platform layer and
// the game layer, bypassing the input event pipeline where needed.
package hooks

// OSKButtonHook is a direct callback from the fbdev platform's button
// handler to the game's on-screen keyboard. The platform calls it on
// button press while it is non-nil; the game sets it when the keyboard
// opens and clears it on close. Values: 0=A, 1=B, 2=START, 3=SELECT.
var OSKButtonHook func(button int) = nil // 0=A, 1=B, 2=START, 3=SELECT, 4=L1, 5=R1
