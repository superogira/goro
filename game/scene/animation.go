package scene

import (
	"time"
)

type ActorAnimation struct {
	ActionFamily   int
	Started        time.Time
	Duration       time.Duration
	StartDelay     time.Duration
	Loop           bool
	PingPong       bool
	Play           bool
	HasPlay        bool
	Length         int
	HasLength      bool
	FrameOffset    int
	HasFrameOffset bool
	HoldFinal      bool
	FixedMotion    int
	HasFixedMotion bool
	Speed          time.Duration
	HasSpeed       bool
	AutoCounter    bool
	Casting        bool
	Combo          bool
	KeepMovement   bool
	Next           *ActorAnimation
}
