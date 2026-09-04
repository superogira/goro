// soundprobe prints the ACT sound table of a monster, resolving its sprite
// the same way the game does (job -> npcidentity resource name -> candidate
// paths). Usage: soundprobe <data-dir> <job-id> [job-id...]
package main

import (
	"fmt"
	"os"

	"github.com/kivutar/goro/audio"
	"github.com/kivutar/goro/res"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: soundprobe <data-dir> <job-id> [job-id...]")
		os.Exit(1)
	}
	manager, err := res.NewManager(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "manager:", err)
		os.Exit(1)
	}
	for _, arg := range os.Args[2:] {
		var job int
		if _, err := fmt.Sscanf(arg, "%d", &job); err != nil {
			fmt.Fprintln(os.Stderr, "bad job id:", arg)
			continue
		}
		name, ok := manager.NonPCResourceName(job)
		if !ok {
			fmt.Printf("job %d: no resource name\n", job)
			continue
		}
		loaded := false
		for _, candidate := range res.NonPCSpriteResourceCandidates(job, name, "act") {
			data, err := manager.ReadFile(candidate)
			if err != nil {
				continue
			}
			act, err := res.ParseACT(data)
			if err != nil {
				fmt.Printf("job %d (%s): act parse error: %v\n", job, name, err)
				loaded = true
				break
			}
			loaded = true
			fmt.Printf("job %d (%s): %d sounds", job, name, len(act.Sounds))
			for i, sound := range act.Sounds {
				fmt.Printf("\n  [%d] %q -> %v", i, sound, audio.SFXPathCandidates(sound))
			}
			fmt.Println()
			break
		}
		if !loaded {
			fmt.Printf("job %d (%s): act not found\n", job, name)
		}
	}
}
