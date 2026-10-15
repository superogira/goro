package session

import "time"

type SuperNovicePrayer struct {
	lines [11]string
	times [11]time.Time
	count int
}

func (p *SuperNovicePrayer) Record(message string, now time.Time) {
	if p.count == 10 {
		copy(p.lines[:], p.lines[1:])
		copy(p.times[:], p.times[1:])
	}
	p.lines[p.count] = message
	p.times[p.count] = now
	p.count = min(p.count+1, 10)
}

func (p *SuperNovicePrayer) Matches(first, last string, now time.Time) bool {
	if p.count < 10 || first == "" || last == "" {
		return false
	}
	elapsed := now.Sub(p.times[0])
	if elapsed > 0 && elapsed < 10*time.Second {
		return false
	}
	// Sakexe checks slots 7 and 9, ignoring the middle line. Once full, slot
	// 10 holds the current message; on the first fill, slot 9 is still current.
	return p.lines[7] == first && p.lines[9] == last
}
