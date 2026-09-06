package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/kivutar/goro/res"
)

func main() {
	g, err := res.OpenGRF(os.Args[1])
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	names := g.Names()
	fmt.Println("entries:", len(names))
	sort.Strings(names)
	limit := 40
	if len(names) < limit {
		limit = len(names)
	}
	for i := 0; i < limit; i++ {
		fmt.Println(" ", names[i])
	}
	// check specific paths
	for _, q := range os.Args[2:] {
		fmt.Printf("has %s: %v\n", q, g.Has(q))
	}
}
