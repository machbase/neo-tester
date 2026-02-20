package stockbench

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machgo"
)

var nFetch = 100
var host = "192.168.0.90"
var port = 51000
var user = "sys"
var password = "manager"
var code = "WISH"

// goos: darwin
// goarch: arm64
// pkg: tester/stockbench
// cpu: Apple M5
//
// "github.com/machbase/neo-server/v8/api/machcli"
// BenchmarkSelect-10    	      86	  14861460 ns/op	   25010 B/op	    1543 allocs/op
//
// "github.com/machbase/neo-server/v8/api/machgo"
// BenchmarkSelect-10    	     228	   4730629 ns/op	   30324 B/op	     475 allocs/op

func BenchmarkSelect(b *testing.B) {
	ctx := context.Background()
	db, err := machgo.NewDatabase(&machgo.Config{
		Host:         host,
		Port:         port,
		MaxOpenConn:  -1,
		MaxOpenQuery: -1,
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var conn *machgo.Conn
	if c, err := db.Connect(ctx, api.WithPassword(user, password)); err != nil {
		panic(err)
	} else {
		conn = c.(*machgo.Conn)
		defer conn.Close()
	}

	timeTo := time.Now().Add(-time.Duration(2 * time.Minute))
	timeFrom := timeTo.Add(-time.Duration(60 * time.Minute))
	for i := 0; i < b.N; i++ {
		r, err := conn.Query(ctx, `
			select /*+ SCAN_FORWARD(stock_rollup_1m) */ code,
				time,
				sum(sum_price) / sum(cnt) as avg_price,
				sum(sum_volume) as total_volume,
				sum(sum_bid) / sum(cnt) as avg_bid,
				sum(sum_ask) / sum(cnt) as avg_ask
			from stock_rollup_1m
			where code = ?
			and time between ? and ?
			group by code, time
			order by time
			limit ?`, code, timeFrom, timeTo, nFetch)
		if err != nil {
			panic(err)
		}
		rows := r.(*machgo.Rows)
		n := 0
		var name string
		var t time.Time
		var avgPrice float64
		var totalVolume float64
		var avgBid float64
		var avgAsk float64
		for rows.Next() {
			if err := rows.Err(); err != nil {
				panic(err)
			}
			n++
			if err := rows.Scan(&name, &t, &avgPrice, &totalVolume, &avgBid, &avgAsk); err != nil {
				panic(err)
			}
			if name != code {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		err = rows.Close()
	}
}
