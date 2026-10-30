package session

import "testing"

func TestInstanceDestroyEligibilityAndCharacterReset(t *testing.T) {
	s := New()
	s.AccountID = 7
	s.Party.Members = []PartyMember{{AccountID: 7, Role: 0}}
	for _, test := range []struct {
		instance Instance
		phase    InstancePhase
		allowed  bool
	}{
		{Instance{}, InstanceNone, false},
		{Instance{Name: "Tower", Queued: true, QueuePosition: 3}, InstanceWaiting, true},
		{Instance{Name: "Tower", IdleExpiresAt: 1800000300}, InstanceReady, true},
		{Instance{Name: "Tower", ExpiresAt: 1800007200}, InstanceInProgress, false},
	} {
		s.Instance = test.instance
		if s.Instance.Phase() != test.phase || s.CanDestroyInstance() != test.allowed {
			t.Fatalf("instance %+v: phase %v, allowed %v", s.Instance, s.Instance.Phase(), s.CanDestroyInstance())
		}
	}
	s.Instance = Instance{Name: "Tower", Queued: true}
	s.Party.Members[0].Role = 1
	if s.CanDestroyInstance() {
		t.Fatal("member can destroy")
	}
	s.Party.Members[0].Role = 0
	s.Dead = true
	if s.CanDestroyInstance() {
		t.Fatal("dead leader can destroy")
	}
	s.SelectCharacter(Character{ID: 10})
	if s.Instance != (Instance{}) {
		t.Fatal("instance survived character selection")
	}
}
