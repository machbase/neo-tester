package main

import (
	"context"
	"fmt"
	"time"

	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machgo"
)

var host = "127.0.0.1"
var port = 5656
var user = "sys"
var password = "manager"

func main() {
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

	r, err := conn.Query(ctx, `select /*+ SCAN_FORWARD(stock_tick) */ * from stock_tick`)
	if err != nil {
		panic(err)
	}
	rows := r.(*machgo.Rows)
	n := 0

	var code string
	var tm time.Time
	var price float64
	var volume float64
	var bidPrice float64
	var askPrice float64
	for rows.Next() {
		if err := rows.Err(); err != nil {
			panic(err)
		}
		n++
		if err := rows.Scan(&code, &tm, &price, &volume, &bidPrice, &askPrice); err != nil {
			panic(err)
		}
		// if n%100000 == 0 {
		// 	fmt.Printf("%d rows...\n", n)
		// }
		// fmt.Printf("%s %s %f %f %f %f\n", code, tm.Format(time.RFC3339), price, volume, bidPrice, askPrice)
	}
	fmt.Println("Total rows:", n)
	if err := rows.Err(); err != nil {
		panic(err)
	}
	err = rows.Close()
	if err != nil {
		panic(err)
	}
}
