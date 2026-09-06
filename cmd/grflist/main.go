// Command grflist inspects a GRF archive: prints the entry count and either
// the first entries, a per-directory histogram (-stats), or membership of
// specific paths passed as extra arguments.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kivutar/goro/res"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: grflist <file.grf> [-stats | path ...]")
		os.Exit(2)
	}
	g, err := res.OpenGRF(os.Args[1])
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	names := g.Names()
	fmt.Println("entries:", len(names))
	sort.Strings(names)

	args := os.Args[2:]
	if len(args) > 0 && args[0] == "-stats" {
		counts := make(map[string]int)
		for _, n := range names {
			parts := strings.Split(n, "/")
			depth := len(parts) - 1
			if depth > 3 {
				depth = 3
			}
			counts[strings.Join(parts[:depth], "/")]++
		}
		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%6d  %s\n", counts[k], k)
		}
		return
	}

	if len(args) == 0 {
		limit := len(names)
		if limit > 40 {
			limit = 40
		}
		for i := 0; i < limit; i++ {
			fmt.Println(" ", names[i])
		}
		return
	}
	for _, q := range args {
		fmt.Printf("has %s: %v\n", q, g.Has(q))
	}
}
