package stockbench

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machcli"
	"github.com/machbase/neo-server/v8/api/machgo"
)

var nFetch = 100
var host = "192.168.0.90"
var port = 51000
var user = "sys"
var password = "manager"
var code = "WISH"

func BenchmarkSelect_MachCli(b *testing.B) {
	ctx := context.Background()
	conn, err := connectMachCli(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	benchSelect(b, ctx, conn)
}

func BenchmarkSelect_MachGo(b *testing.B) {
	ctx := context.Background()
	conn, err := connectMachGo(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	benchSelect(b, ctx, conn)
}

func BenchmarkSelectRollup_MachCli(b *testing.B) {
	ctx := context.Background()
	conn, err := connectMachCli(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	benchSelectRollup(b, ctx, conn)
}

func BenchmarkSelectRollup_MachGo(b *testing.B) {
	ctx := context.Background()
	conn, err := connectMachGo(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	benchSelectRollup(b, ctx, conn)
}

func connectMachGo(ctx context.Context) (api.Conn, error) {
	db, err := machgo.NewDatabase(&machgo.Config{
		Host:           host,
		Port:           port,
		MaxOpenConn:    -1,
		MaxOpenQuery:   -1,
		StatementReuse: machgo.StatementReuseAuto,
		FetchRows:      1000,
	})
	if err != nil {
		panic(err)
	}

	if c, err := db.Connect(ctx, api.WithPassword(user, password)); err != nil {
		panic(err)
	} else {
		return c, nil
	}
}

func connectMachCli(ctx context.Context) (api.Conn, error) {
	db, err := machcli.NewDatabase(&machcli.Config{
		Host:         host,
		Port:         port,
		MaxOpenConn:  -1,
		MaxOpenQuery: -1,
	})
	if err != nil {
		panic(err)
	}

	if c, err := db.Connect(ctx, api.WithPassword(user, password)); err != nil {
		panic(err)
	} else {
		return c, nil
	}
}

func benchSelectRollup(b *testing.B, ctx context.Context, conn api.Conn) {
	timeTo := time.Now().Add(-time.Duration(2 * time.Minute))
	timeFrom := timeTo.Add(-time.Duration(60 * time.Minute))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := conn.Query(ctx, `
			select 
				code,
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

func benchSelect(b *testing.B, ctx context.Context, conn api.Conn) {
	timeTo := time.Now().Add(-time.Duration(2 * time.Minute))
	timeFrom := timeTo.Add(-time.Duration(60 * time.Second))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := conn.Query(ctx, `
			select
				code,
				time,
				price,
				volume,
				bid_price,
				ask_price
			from stock_tick
			where code = ?
			and time between ? and ?
			order by time
			limit ?`, code, timeFrom, timeTo, nFetch)
		if err != nil {
			panic(err)
		}
		n := 0
		var name string
		var t time.Time
		var price float64
		var volume float64
		var bidPrice float64
		var askPrice float64
		for rows.Next() {
			if err := rows.Err(); err != nil {
				panic(err)
			}
			n++
			if err := rows.Scan(&name, &t, &price, &volume, &bidPrice, &askPrice); err != nil {
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
