package main

import "testing"

func TestParseTableName(t *testing.T) {
	for _, tc := range []struct {
		table, user, owner, name string
	}{
		{"stock_tick", "sys", "SYS", "STOCK_TICK"},
		{"demo.stock_tick", "sys", "DEMO", "STOCK_TICK"},
		{"stock_tick", "demo", "DEMO", "STOCK_TICK"},
	} {
		t.Run(tc.table+"/"+tc.user, func(t *testing.T) {
			owner, name, err := parseTableName(tc.table, tc.user)
			if err != nil || owner != tc.owner || name != tc.name {
				t.Fatalf("got (%q, %q, %v), want (%q, %q, nil)", owner, name, err, tc.owner, tc.name)
			}
		})
	}
	for _, table := range []string{"", ".stock_tick", "demo.", "a.b.c", "demo.stock_tick;drop", "demo.stock tick"} {
		t.Run("invalid/"+table, func(t *testing.T) {
			if _, _, err := parseTableName(table, "sys"); err == nil {
				t.Fatalf("accepted invalid table %q", table)
			}
		})
	}
}
