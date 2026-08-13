package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/machbase/neo-client"
)

func BenchmarkConn(b *testing.B) {
	db, err := sql.Open("machbase", "host=127.0.0.1; port=5656; user=sys; password=manager")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		conn, err := db.Conn(ctx)
		if err != nil {
			panic(err)
		}
		conn.Close()
	}
}

func BenchmarkQuery(b *testing.B) {
	db, err := sql.Open("machbase", "host=127.0.0.1; port=5656; user=sys; password=manager")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	tagName := "tag1"
	for i := 0; i < b.N; i++ {
		rows, err := conn.QueryContext(ctx, "SELECT * FROM tag WHERE name=? LIMIT 100", tagName)
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
