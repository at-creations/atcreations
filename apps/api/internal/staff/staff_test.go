package staff

import "testing"

func TestGuardTarget(t *testing.T) {
	cases := []struct {
		name             string
		actor, target    string
		isAdmin, removes bool
		admins           int
		wantErr          bool
	}{
		{"self", "u1", "u1", false, false, 3, true},
		{"demote last admin", "u1", "u2", true, true, 1, true},
		{"demote one of two admins", "u1", "u2", true, true, 2, false},
		{"promote staff", "u1", "u2", false, false, 1, false},
		{"ban non-admin", "u1", "u2", false, true, 1, false},
	}
	for _, c := range cases {
		err := GuardTarget(c.actor, c.target, c.isAdmin, c.removes, c.admins)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", c.name, err, c.wantErr)
		}
	}
}
