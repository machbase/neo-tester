package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machcli"
)

func BenchmarkConn(b *testing.B) {
	db, err := machcli.NewDatabase(&machcli.Config{
		Host:         "127.0.0.1",
		Port:         5656,
		MaxOpenConn:  -1,
		MaxOpenQuery: -1,
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		conn, err := db.Connect(ctx, api.WithPassword("sys", "manager"))
		if err != nil {
			panic(err)
		}
		conn.Close()
	}
}

func BenchmarkQuery(b *testing.B) {
	db, err := machcli.NewDatabase(&machcli.Config{
		Host:         "127.0.0.1",
		Port:         5656,
		MaxOpenConn:  -1,
		MaxOpenQuery: -1,
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()
	ctx := context.Background()
	conn, err := db.Connect(ctx, api.WithPassword("sys", "manager"))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for i := 0; i < b.N; i++ {
		rows, err := conn.Query(ctx, "SELECT * FROM tag WHERE name='tag1' LIMIT 100")
		if err != nil {
			panic(err)
		}
		for rows.Next() {
			var name string
			var t time.Time
			var v float64
			if err := rows.Scan(&name, &t, &v); err != nil {
				panic(err)
			}
			if name != "tag1" {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		rows.Close()
	}
}
