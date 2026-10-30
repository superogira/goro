package session

type InstancePhase uint8

const (
	InstanceNone InstancePhase = iota
	InstanceWaiting
	InstanceReady
	InstanceInProgress
)

// Instance is server-owned and survives map changes. Deadlines are Unix seconds;
// zero means the corresponding timer is not running.
type Instance struct {
	Name          string
	Queued        bool
	QueuePosition int16
	ExpiresAt     uint32
	IdleExpiresAt uint32
}

func (i Instance) Phase() InstancePhase {
	switch {
	case i.Name == "":
		return InstanceNone
	case i.Queued:
		return InstanceWaiting
	case i.IdleExpiresAt != 0:
		return InstanceReady
	default:
		return InstanceInProgress
	}
}

func (s *Session) CanDestroyInstance() bool {
	if s == nil || s.Dead || s.Instance.Phase() != InstanceWaiting && s.Instance.Phase() != InstanceReady {
		return false
	}
	for _, member := range s.Party.Members {
		if member.AccountID == s.AccountID {
			return member.Leader()
		}
	}
	return false
}
