package companion

import (
	"testing"

	"github.com/kivutar/goro/db"
)

func TestMercenaryWeaponBaseJobUsesGuildHumanoidResources(t *testing.T) {
	cases := []struct {
		name string
		job  int
		want int
	}{
		{name: "archer", job: 6017, want: db.JobArcher},
		{name: "lancer", job: 6027, want: db.JobSwordman},
		{name: "sword", job: 6037, want: db.JobSwordman},
		{name: "unknown", job: 9999, want: db.JobNovice},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MercenaryWeaponBaseJob(tc.job); got != tc.want {
				t.Fatalf("weapon base job = %d, want %d", got, tc.want)
			}
		})
	}
}
