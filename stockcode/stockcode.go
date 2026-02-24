package main

import (
	"context"
	"flag"
	"time"

	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machcli"
)

var host = "127.0.0.1"
var port = 5656
var user = "sys"
var password = "manager"
var code = "AAPL"

func main() {
	flag.StringVar(&host, "h", host, "server host")
	flag.IntVar(&port, "p", port, "server port")
	flag.StringVar(&user, "u", user, "user")
	flag.StringVar(&password, "P", password, "password")
	flag.StringVar(&code, "code", code, "stock code (tag) to insert/query")
	flag.Parse()

	db, err := machcli.NewDatabase(&machcli.Config{
		Host:         host,
		Port:         port,
		MaxOpenConn:  -1,
		MaxOpenQuery: -1,
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	conn, err := db.Connect(ctx, api.WithPassword(user, password))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	rows, err := conn.Query(ctx, `
	select
		DATE_TRUNC('minute', time) as mtime,
		sum(sum_price) / sum(cnt) as avg_price,
		sum(sum_volume) as total_volume,
		sum(sum_bid) / sum(cnt) as avg_bid,
		sum(sum_ask) / sum(cnt) as avg_ask
	from stock_rollup_1m
	where code = ?
	and time >= date_trunc('minute', sysdate) - 1h
	and time < date_trunc('minute', sysdate) - 2m
	group by mtime
	order by mtime
	UNION ALL
	select
		DATE_TRUNC('minute', time) as mtime,
		SUM(price)/count(*) as avg_price,
		SUM(volume) as total_volume,
		SUM(bid_price)/count(*) as avg_bid,
		SUM(ask_price)/count(*) as avg_ask
	from stock_tick
	where code = ?
	and time >= date_trunc('minute', sysdate) - 2m
	group by mtime
	order by mtime
	`, code, code)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var mtime time.Time
		var avgPrice, totalVolume, avgBid, avgAsk float64
		if err := rows.Scan(&mtime, &avgPrice, &totalVolume, &avgBid, &avgAsk); err != nil {
			panic(err)
		}
		println(mtime.String(), avgPrice, totalVolume, avgBid, avgAsk)
	}
}
