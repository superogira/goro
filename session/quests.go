package session

import "slices"

type QuestObjective struct {
	MonsterID uint32
	Name      string
	Current   uint16
	// Required is unknown (zero) until the server sends a hunt update.
	Required uint16
}

type Quest struct {
	ID         uint32
	Active     bool
	ExpiresAt  uint32
	Objectives []QuestObjective
}

// Quests is character-owned. Map changes keep it; a fresh server list replaces it.
// Version lets closed/idle journal windows avoid scanning or rebuilding rows.
type Quests struct {
	Entries map[uint32]Quest
	Version uint64
}

func (q *Quests) Replace(entries []Quest) {
	q.Entries = make(map[uint32]Quest, len(entries))
	for _, entry := range entries {
		entry.Objectives = slices.Clone(entry.Objectives)
		q.Entries[entry.ID] = entry
	}
	q.Version++
}

func (q *Quests) Set(entry Quest) {
	if q.Entries == nil {
		q.Entries = make(map[uint32]Quest)
	}
	entry.Objectives = slices.Clone(entry.Objectives)
	q.Entries[entry.ID] = entry
	q.Version++
}

func (q *Quests) SetMission(id, expiresAt uint32, objectives []QuestObjective) {
	entry, ok := q.Entries[id]
	if !ok {
		entry = Quest{ID: id, Active: true}
	}
	entry.ExpiresAt, entry.Objectives = expiresAt, objectives
	q.Set(entry)
}

func (q *Quests) Remove(id uint32) {
	if _, ok := q.Entries[id]; ok {
		delete(q.Entries, id)
		q.Version++
	}
}

func (q *Quests) SetActive(id uint32, active bool) {
	if entry, ok := q.Entries[id]; ok && entry.Active != active {
		entry.Active = active
		q.Entries[id] = entry
		q.Version++
	}
}

func (q *Quests) UpdateHunt(id, monsterID uint32, current, required uint16) {
	entry, ok := q.Entries[id]
	if !ok {
		return
	}
	for i, objective := range entry.Objectives {
		if objective.MonsterID == monsterID && (objective.Current != current || objective.Required != required) {
			entry.Objectives[i].Current, entry.Objectives[i].Required = current, required
			q.Version++
		}
	}
}
