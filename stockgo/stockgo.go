package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machgo"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var nClient = 50
var nCount = 1000
var nFetch = 100
var doProfile = false
var doPreparedStmt = false
var doRollupQuery = false
var doUnionQuery = false
var doReuseStmt = false
var sessionElapsed []time.Duration
var sessionReadBytes []uint64
var sessionWrittenBytes []uint64
var host = "127.0.0.1"
var port = 5656
var user = "sys"
var password = "manager"
var code = "AAPL"

func main() {
	flag.IntVar(&nClient, "c", nClient, "number of clients")
	flag.IntVar(&nCount, "n", nCount, "number of queries per client")
	flag.IntVar(&nFetch, "f", nFetch, "number of rows to fetch per query")
	flag.StringVar(&host, "h", host, "server host")
	flag.IntVar(&port, "p", port, "server port")
	flag.StringVar(&user, "u", user, "user")
	flag.StringVar(&password, "P", password, "password")
	flag.BoolVar(&doPreparedStmt, "prep", doPreparedStmt, "use prepared statement")
	flag.BoolVar(&doRollupQuery, "rollup", doRollupQuery, "perform rollup query instead of tick query")
	flag.BoolVar(&doUnionQuery, "union", doUnionQuery, "perform union query instead of tick query")
	flag.StringVar(&code, "code", code, "stock code (tag) to insert/query")
	flag.BoolVar(&doProfile, "prof", doProfile, "enable profiling")
	flag.BoolVar(&doReuseStmt, "reuse", doReuseStmt, "reuse prepared statement")
	flag.Parse()

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

	ctx := context.Background()
	sessionElapsed = make([]time.Duration, nClient)
	sessionReadBytes = make([]uint64, nClient)
	sessionWrittenBytes = make([]uint64, nClient)
	var startCh = make(chan struct{})
	var wg sync.WaitGroup

	if doProfile {
		// go tool pprof -http=:8080 /tmp/cpu.prof
		cpu_prof, err := os.Create("/tmp/cpu.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(cpu_prof)
		defer pprof.StopCPUProfile()

		// go tool pprof -http=:8080 /tmp/mem.prof
		mem_prof, err := os.Create("/tmp/mem.prof")
		if err != nil {
			panic(err)
		}
		defer func() {
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(mem_prof); err != nil {
				panic(err)
			}
			mem_prof.Close()
		}()
	}

	var start = time.Now()
	for i := 0; i < nClient; i++ {
		wg.Add(1)

		go func(ctx context.Context, clientId int) {
			defer wg.Done()
			<-startCh
			var conn *machgo.Conn
			var options = []api.ConnectOption{
				api.WithPassword(user, password),
				api.WithIOMetrics(true),
				api.WithStatementCache(api.StatementCacheAuto),
			}
			if doReuseStmt {
				options = append(options, api.WithStatementCache(api.StatementCacheAuto))
			}
			if c, err := db.Connect(ctx, options...); err != nil {
				panic(err)
			} else {
				conn = c.(*machgo.Conn)
			}
			defer func() {
				err := conn.Close()
				if err != nil {
					panic(err)
				}
			}()
			clientStart := time.Now()
			defer func() {
				elapsed := time.Since(clientStart)
				sessionElapsed[clientId] = elapsed
				readBytes, writtenBytes, enabled := conn.IOMetrics()
				if enabled {
					sessionReadBytes[clientId] = readBytes
					sessionWrittenBytes[clientId] = writtenBytes
				}
			}()

			if doRollupQuery {
				timeTo := time.Now().Add(-time.Duration(2 * time.Minute))
				timeFrom := timeTo.Add(-time.Duration(60 * time.Minute))
				if doPreparedStmt {
					RunRollupPreparedQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: timeFrom, betweenTo: timeTo})
				} else {
					RunRollupQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: timeFrom, betweenTo: timeTo})
				}
			} else if doUnionQuery {
				timeTo := time.Now()
				timeFrom := timeTo.Add(-time.Duration(60 * time.Minute))
				RunUnionQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: timeFrom, betweenTo: timeTo})
			} else {
				timeTo := time.Now()
				timeFrom := timeTo.Add(-time.Duration(1 * time.Minute))
				if doPreparedStmt {
					RunPreparedQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: timeFrom, betweenTo: timeTo})
				} else {
					RunQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: timeFrom, betweenTo: timeTo})
				}
			}
		}(ctx, i)
	}
	close(startCh)
	wg.Wait()

	mode := "Query"
	if doPreparedStmt {
		mode = "Prepare"
	}
	fmt.Printf("All clients (%d) query(%d) (%s mode) completed in %v  %d ops/sec\n",
		nClient, nCount, mode, time.Since(start), int(float64(nClient*nCount)/time.Since(start).Seconds()))
	var totalSessionElapsed time.Duration
	var minSessionElapsed time.Duration
	var maxSessionElapsed time.Duration
	for i, d := range sessionElapsed {
		totalSessionElapsed += d
		if i == 0 || minSessionElapsed > d {
			minSessionElapsed = d
		}
		if maxSessionElapsed < d {
			maxSessionElapsed = d
		}
	}
	avgSessionElapsed := time.Duration(int64(totalSessionElapsed) / int64(nClient))
	fmt.Printf("  Sessions: min %v, max %v, avg %v\n", minSessionElapsed, maxSessionElapsed, avgSessionElapsed)

	var totalReadBytes uint64
	var totalWrittenBytes uint64
	var countIOMetrics int
	for i := 0; i < nClient; i++ {
		if sessionReadBytes[i] > 0 || sessionWrittenBytes[i] > 0 {
			countIOMetrics++
			totalReadBytes += sessionReadBytes[i]
			totalWrittenBytes += sessionWrittenBytes[i]
		}
	}
	avgReadBytesPerSec := uint64(0)
	avgWrittenBytesPerSec := uint64(0)
	if countIOMetrics > 0 {
		avgReadBytesPerSec = totalReadBytes / uint64(totalSessionElapsed.Seconds())
		avgWrittenBytesPerSec = totalWrittenBytes / uint64(totalSessionElapsed.Seconds())
	}
	if countIOMetrics > 0 {
		fmt.Printf("  IO Bytes: total read %s (%s/s), total written %s (%s/s)\n",
			Bytes(int64(totalReadBytes)), Bytes(int64(avgReadBytesPerSec)),
			Bytes(int64(totalWrittenBytes)), Bytes(int64(avgWrittenBytesPerSec)))
	} else {
		fmt.Printf("  IO Bytes: not available\n")
	}
}

type Query struct {
	code        string
	nFetch      int
	betweenFrom time.Time
	betweenTo   time.Time
}

func RunQuery(ctx context.Context, clientId int, conn *machgo.Conn, nCount int, q Query) {
	for j := 0; j < nCount; j++ {
		tick := time.Now()
		r, err := conn.Query(ctx, `
			select code,
				time,
				price,
				volume,
				bid_price,
				ask_price
			from stock_tick
			where code = ?
			and time between ? and ?
			order by time
			limit ?`, q.code, q.betweenFrom, q.betweenTo, q.nFetch)
		if err != nil {
			fmt.Printf("Query error(1), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
		rows := r.(*machgo.Rows)
		n := 0
		for rows.Next() {
			if err := rows.Err(); err != nil {
				panic(err)
			}
			n++
			var name string
			var t time.Time
			var avgPrice float64
			var totalVolume float64
			var avgBid float64
			var avgAsk float64
			if err := rows.Scan(&name, &t, &avgPrice, &totalVolume, &avgBid, &avgAsk); err != nil {
				panic(err)
			}
			if name != q.code {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		tick = time.Now()
		err = rows.Close()
		if err != nil {
			fmt.Printf("Close error(2), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
	}
}

func RunPreparedQuery(ctx context.Context, clientId int, conn api.Conn, nCount int, q Query) {
	var stmt *machgo.PreparedStmt
	if s, err := conn.Prepare(ctx, `
			select /*+ SCAN_FORWARD(stock_tick) */ code,
				time,
				price,
				volume,
				bid_price,
				ask_price
			from stock_tick
			where code = ?
			and time between ? and ?
			order by time
			limit ?`); err != nil {
		panic(err)
	} else {
		stmt = s.(*machgo.PreparedStmt)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			panic(err)
		}
	}()
	for j := 0; j < nCount; j++ {
		tick := time.Now()
		r, err := stmt.Query(ctx, q.code, q.betweenFrom, q.betweenTo, q.nFetch)
		if err != nil {
			fmt.Printf("Query error(2), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
		rows := r.(*machgo.Rows)
		n := 0
		for rows.Next() {
			if err := rows.Err(); err != nil {
				panic(err)
			}
			n++
			var name string
			var t time.Time
			var avgPrice float64
			var totalVolume float64
			var avgBid float64
			var avgAsk float64
			if err := rows.Scan(&name, &t, &avgPrice, &totalVolume, &avgBid, &avgAsk); err != nil {
				panic(err)
			}
			if name != q.code {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		tick = time.Now()
		if err = rows.Close(); err != nil {
			fmt.Printf("Close error(3), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
	}
}

func RunRollupQuery(ctx context.Context, clientId int, conn *machgo.Conn, nCount int, q Query) {
	for j := 0; j < nCount; j++ {
		tick := time.Now()
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
			limit ?`, q.code, q.betweenFrom, q.betweenTo, q.nFetch)
		if err != nil {
			fmt.Printf("Query error(1), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
		rows := r.(*machgo.Rows)
		n := 0
		for rows.Next() {
			if err := rows.Err(); err != nil {
				panic(err)
			}
			n++
			var name string
			var t time.Time
			var avgPrice float64
			var totalVolume float64
			var avgBid float64
			var avgAsk float64
			if err := rows.Scan(&name, &t, &avgPrice, &totalVolume, &avgBid, &avgAsk); err != nil {
				panic(err)
			}
			if name != q.code {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		tick = time.Now()
		err = rows.Close()
		if err != nil {
			fmt.Printf("Close error(2), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
	}
}

func RunRollupPreparedQuery(ctx context.Context, clientId int, conn api.Conn, nCount int, q Query) {
	var stmt *machgo.PreparedStmt
	if s, err := conn.Prepare(ctx, `
			select code,
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
			limit ?`); err != nil {
		panic(err)
	} else {
		stmt = s.(*machgo.PreparedStmt)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			panic(err)
		}
	}()
	for j := 0; j < nCount; j++ {
		tick := time.Now()
		r, err := stmt.Query(ctx, q.code, q.betweenFrom, q.betweenTo, q.nFetch)
		if err != nil {
			fmt.Printf("Query error(2), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
		rows := r.(*machgo.Rows)
		n := 0
		for rows.Next() {
			if err := rows.Err(); err != nil {
				panic(err)
			}
			n++
			var name string
			var t time.Time
			var avgPrice float64
			var totalVolume float64
			var avgBid float64
			var avgAsk float64
			if err := rows.Scan(&name, &t, &avgPrice, &totalVolume, &avgBid, &avgAsk); err != nil {
				panic(err)
			}
			if name != q.code {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		tick = time.Now()
		if err = rows.Close(); err != nil {
			fmt.Printf("Close error(3), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
	}
}

var (
	defaultLang language.Tag = language.English
)

func Bytes(v int64) string {
	p := message.NewPrinter(defaultLang)
	f := float64(v)
	u := ""
	switch {
	case v >= 1024*1024*1024*1024:
		f = f / (1024 * 1024 * 1024 * 1024)
		u = "TB"
	case v >= 1024*1024*1024:
		f = f / (1024 * 1024 * 1024)
		u = "GB"
	case v >= 1024*1024:
		f = f / (1024 * 1024)
		u = "MB"
	case v >= 1024:
		f = f / 1024
		u = "KB"
	default:
		return p.Sprintf("%dB", v)
	}
	return p.Sprintf("%.1f%s", f, u)
}

func RunUnionQuery(ctx context.Context, clientId int, conn *machgo.Conn, nCount int, q Query) {
	for j := 0; j < nCount; j++ {
		tick := time.Now()
		r, err := conn.Query(ctx, `
			select
				DATE_TRUNC('minute', time) as mtime,
				sum(sum_price) / sum(cnt) as avg_price,
				sum(sum_volume) as total_volume,
				sum(sum_bid) / sum(cnt) as avg_bid,
				sum(sum_ask) / sum(cnt) as avg_ask
			from stock_rollup_1m
			where code = ?
			and time >= date_trunc('minute', sysdate) - 60m
			and time < date_trunc('minute', sysdate) - 2m
			group by mtime
			order by mtime
			UNION ALL
			select
				DATE_TRUNC('minute', time) as mtime,
				AVG(price) as avg_price,
				SUM(volume) as total_volume,
				AVG(bid_price) as avg_bid,
				AVG(ask_price) as avg_ask
			from stock_tick
			where code = ?
			and time >= date_trunc('minute', sysdate) - 2m
			group by mtime
			order by mtime`,
			q.code, q.code)
		if err != nil {
			fmt.Printf("Query error(1), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
		rows := r.(*machgo.Rows)
		n := 0
		for rows.Next() {
			if err := rows.Err(); err != nil {
				panic(err)
			}
			n++
			var mtime time.Time
			var avgPrice float64
			var totalVolume float64
			var avgBid float64
			var avgAsk float64
			if err := rows.Scan(&mtime, &avgPrice, &totalVolume, &avgBid, &avgAsk); err != nil {
				panic(err)
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		tick = time.Now()
		err = rows.Close()
		if err != nil {
			fmt.Printf("Close error(2), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
	}
}
