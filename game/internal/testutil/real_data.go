package testutil

import (
	"os"
	"testing"

	"github.com/kivutar/goro/res"
)

func RealDataManager(t *testing.T) *res.Manager {
	t.Helper()
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to run against real client data")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() {
			_ = archive.Close()
		})
	}
	return manager
}
