package session

import "testing"

func TestQuestLifecycle(t *testing.T) {
	var quests Quests
	quests.Replace([]Quest{{ID: 1001}, {ID: 2002, Active: true}})
	quests.SetMission(1001, 1800000000, []QuestObjective{{MonsterID: 1002, Name: "Poring", Current: 3}})
	if quests.Entries[1001].Active {
		t.Fatal("mission reactivated an inactive quest")
	}
	quests.UpdateHunt(1001, 1002, 4, 10)
	objective := quests.Entries[1001].Objectives[0]
	if objective.Current != 4 || objective.Required != 10 {
		t.Fatalf("objective = %+v", objective)
	}
	version := quests.Version
	quests.UpdateHunt(1001, 1002, 4, 10)
	quests.SetActive(1001, false)
	quests.UpdateHunt(1001, 9999, 1, 1)
	quests.UpdateHunt(9999, 1002, 1, 1)
	quests.SetActive(9999, true)
	quests.Remove(9999)
	if quests.Version != version || len(quests.Entries) != 2 {
		t.Fatal("no-op updates changed the journal")
	}
	quests.UpdateHunt(1001, 1002, 0, 10)
	if quests.Entries[1001].Objectives[0].Current != 0 {
		t.Fatal("zero progress was ignored")
	}
	quests.SetActive(1001, true)
	if !quests.Entries[1001].Active {
		t.Fatal("activation ACK was ignored")
	}
	quests.Remove(1001)
	if _, ok := quests.Entries[1001]; ok {
		t.Fatal("completed quest was retained")
	}
	quests.Replace(nil)
	if len(quests.Entries) != 0 {
		t.Fatal("empty list did not clear old quests")
	}
}

func TestQuestOwnsObjectivesAndResetsOnCharacterSelection(t *testing.T) {
	s := New()
	objectives := []QuestObjective{{MonsterID: 1002, Current: 2}}
	s.Quests.Set(Quest{ID: 1001, Objectives: objectives})
	objectives[0].Current = 99
	if s.Quests.Entries[1001].Objectives[0].Current != 2 {
		t.Fatal("retained caller's objective slice")
	}
	s.Quests.Set(Quest{ID: 1001, Active: true})
	if len(s.Quests.Entries) != 1 || len(s.Quests.Entries[1001].Objectives) != 0 {
		t.Fatal("re-added quest was not replaced")
	}
	s.SelectCharacter(Character{ID: 1})
	if len(s.Quests.Entries) != 0 || s.Quests.Version != 0 {
		t.Fatal("quests leaked between characters")
	}
}
