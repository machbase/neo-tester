package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/machbase/neo-engine/v8/native"
	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machcli"
)

var nClient = 50
var nCount = 1000
var nFetch = 100
var doCpuProfile = false
var doOSThreadLock = false
var doCreateData = false
var doPreparedStmt = false
var doRollupQuery = false
var doAppendDataTPS = float64(0)
var sessionElapsed []time.Duration
var host = "127.0.0.1"
var port = 5656
var user = "sys"
var password = "manager"
var code = "AAPL"
var csvPath = ""
var csvURL = "https://stooq.com/q/d/l/?s=aapl.us&i=d"
var csvCache = filepath.Join(os.TempDir(), "aapl.us.d.csv")
var csvRefresh = false
var csvMaxRows = 0

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
	flag.StringVar(&code, "code", code, "stock code (tag) to insert/query")
	flag.StringVar(&csvPath, "csv", csvPath, "local CSV file path to load when -create")
	flag.StringVar(&csvURL, "csv-url", csvURL, "CSV URL to download when -create (ex: https://stooq.com/q/d/l/?s=aapl.us&i=5)")
	flag.StringVar(&csvCache, "csv-cache", csvCache, "downloaded CSV cache path")
	flag.BoolVar(&csvRefresh, "csv-refresh", csvRefresh, "force re-download CSV even if cache exists")
	flag.IntVar(&csvMaxRows, "csv-max", csvMaxRows, "max rows to load from CSV (0 = all)")
	flag.BoolVar(&doOSThreadLock, "T", doOSThreadLock, "enable OS thread lock")
	flag.BoolVar(&doCpuProfile, "prof", doCpuProfile, "enable cpu profiling")
	flag.BoolVar(&doCreateData, "create", doCreateData, "create initial data")
	flag.Float64Var(&doAppendDataTPS, "append", doAppendDataTPS, "append data in TPS (0 to disable, 5 = 20ms interval)")
	flag.Parse()

	fmt.Println("Neo Client Version:", native.Version, "Build:", native.GitHash)
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
	if doCreateData {
		CreateData(ctx, db)
	}
	if doAppendDataTPS > 0 {
		stopFunc := AppendData(ctx, db, doAppendDataTPS)
		defer stopFunc()
	}
	sessionElapsed = make([]time.Duration, nClient)
	var startCh = make(chan struct{})
	var wg sync.WaitGroup

	if doCpuProfile {
		// go tool pprof -http=:8080 /tmp/query /tmp/cpu.prof
		cpu_prof, err := os.Create("/tmp/cpu.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(cpu_prof)
		defer pprof.StopCPUProfile()
	}

	var start = time.Now()
	for i := 0; i < nClient; i++ {
		wg.Add(1)

		go func(ctx context.Context, clientId int) {
			defer wg.Done()
			if doOSThreadLock {
				runtime.LockOSThread()
			}
			<-startCh
			var conn *machcli.Conn

			if c, err := db.Connect(ctx, api.WithPassword(user, password)); err != nil {
				panic(err)
			} else {
				conn = c.(*machcli.Conn)
			}
			defer func() {
				err := conn.Close()
				if err != nil {
					panic(err)
				}
			}()
			clientStart := time.Now()
			defer func() {
				sessionElapsed[clientId] = time.Since(clientStart)
			}()
			if doRollupQuery {
				if doPreparedStmt {
					RunRollupPreparedQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: "1986-04-10", betweenTo: "1986-04-30"})
				} else {
					RunRollupQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: "1986-04-10", betweenTo: "1986-04-30"})
				}
			} else {
				if doPreparedStmt {
					RunPreparedQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: "1986-04-10", betweenTo: "1986-04-15"})
				} else {
					RunQuery(ctx, clientId, conn, nCount, Query{code: code, nFetch: nFetch, betweenFrom: "1986-04-10", betweenTo: "1986-04-15"})
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
}

func CreateData(ctx context.Context, db api.Database) {
	conn, err := db.Connect(ctx, api.WithPassword(user, password))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	result := conn.Exec(ctx, `create tag table stock_tick (
									code      varchar(20) primary key,
									time      datetime basetime,
									price     double,
									volume    double,
									bid_price double,
									ask_price double
								)`)
	if result.Err() != nil {
		panic(result.Err())
	}
	result = conn.Exec(ctx, `create tag table stock_rollup_1m (
									code      varchar(20) primary key,
									time      datetime basetime,
									sum_price double,
									sum_volume double,
									sum_bid   double,
									sum_ask   double,
									cnt       integer
								)`)
	if result.Err() != nil {
		panic(result.Err())
	}
	result = conn.Exec(ctx, `create rollup rollup_stock_1m
								into (stock_rollup_1m)
								as (
									select code,
											date_trunc('minute', time) as time,
											sum(price) as sum_price,
											sum(volume) as sum_volume,
											sum(bid_price) as sum_bid,
											sum(ask_price) as sum_ask,
											count(*) as cnt
										from stock_tick
									group by code, time
								)
								interval 1 min`)
	if result.Err() != nil {
		panic(result.Err())
	}

	// Load test data
	loaded := 0
	if csvPath != "" || csvURL != "" {
		path, err := ensureCSV(csvPath, csvURL, csvCache, csvRefresh)
		if err != nil {
			panic(err)
		}
		appendConn, err := db.Connect(ctx, api.WithPassword(user, password))
		if err != nil {
			panic(err)
		}
		loaded, err = loadStockCSV(ctx, appendConn, code, path, csvMaxRows)
		if err != nil {
			panic(err)
		}
		appendConn.Close()
		fmt.Printf("Loaded %d rows from %s\n", loaded, path)
	} else {
		// Backward-compatible tiny sample
		data := []struct {
			code     string
			time     string
			price    float64
			volume   float64
			bidPrice float64
			askPrice float64
		}{
			{code, "2026-01-27 09:30:01.123", 150.25, 100, 150.20, 150.30},
			{code, "2026-01-27 09:30:01.456", 150.30, 200, 150.25, 150.35},
			{code, "2026-01-27 09:30:59.999", 150.80, 300, 150.70, 150.90},
		}
		for _, d := range data {
			result = conn.Exec(ctx, `insert into stock_tick values (?, ?, ?, ?, ?, ?)`, d.code, d.time, d.price, d.volume, d.bidPrice, d.askPrice)
			if result.Err() != nil {
				panic(result.Err())
			}
			loaded++
		}
	}
	result = conn.Exec(ctx, `exec rollup_force(rollup_stock_1m)`)
	if result.Err() != nil {
		panic(result.Err())
	}
}

type Query struct {
	code        string
	nFetch      int
	betweenFrom string
	betweenTo   string
}

func RunQuery(ctx context.Context, clientId int, conn *machcli.Conn, nCount int, q Query) {
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
		rows := r.(*machcli.Rows)
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
	var stmt *machcli.PreparedStmt
	if s, err := conn.Prepare(ctx, `
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
			limit ?`); err != nil {
		panic(err)
	} else {
		stmt = s.(*machcli.PreparedStmt)
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
		rows := r.(*machcli.Rows)
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

func RunRollupQuery(ctx context.Context, clientId int, conn *machcli.Conn, nCount int, q Query) {
	for j := 0; j < nCount; j++ {
		tick := time.Now()
		r, err := conn.Query(ctx, `
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
			limit ?`, q.code, q.betweenFrom, q.betweenTo, q.nFetch)
		if err != nil {
			fmt.Printf("Query error(1), client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
		rows := r.(*machcli.Rows)
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
	var stmt *machcli.PreparedStmt
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
		stmt = s.(*machcli.PreparedStmt)
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
		rows := r.(*machcli.Rows)
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

func ensureCSV(localPath string, url string, cachePath string, refresh bool) (string, error) {
	if localPath != "" {
		return localPath, nil
	}
	if url == "" {
		return "", fmt.Errorf("csv-url is empty and no -csv provided")
	}
	if cachePath == "" {
		return "", fmt.Errorf("csv-cache is empty")
	}
	if !refresh {
		if st, err := os.Stat(cachePath); err == nil && st.Size() > 0 {
			return cachePath, nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", err
	}
	if err := downloadFile(url, cachePath); err != nil {
		return "", err
	}
	return cachePath, nil
}

func downloadFile(url string, dst string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "neo-tester/stock")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("download failed: %s, status=%d, body=%q", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func loadStockCSV(ctx context.Context, dbConn api.Conn, code string, path string, maxRows int) (int, error) {
	var conn *machcli.Conn
	if c, ok := dbConn.(*machcli.Conn); !ok {
		return 0, fmt.Errorf("invalid machcli.Conn")
	} else {
		conn = c
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.ReuseRecord = true

	headers, err := r.Read()
	if err != nil {
		return 0, err
	}
	idx := map[string]int{}
	for i, h := range headers {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	// Stooq daily format: Date,Open,High,Low,Close,Volume
	dateIdx, okDate := idx["date"]
	closeIdx, okClose := idx["close"]
	volumeIdx, okVol := idx["volume"]
	if !okDate || !okClose || !okVol {
		return 0, fmt.Errorf("unsupported CSV headers: need Date/Close/Volume, got %v", headers)
	}

	var appender *machcli.Appender
	if a, err := conn.Appender(ctx, "stock_tick"); err != nil {
		panic(err)
	} else {
		appender = a.(*machcli.Appender)
	}

	// code, time, closeVal, volVal, closeVal, closeVal
	appender = appender.WithInputFormats("", "YYYY-MM-DD HH24:MI:SS.mmm").(*machcli.Appender)

	loaded := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return loaded, err
		}
		if dateIdx >= len(rec) || closeIdx >= len(rec) || volumeIdx >= len(rec) {
			continue
		}
		dateStr := strings.TrimSpace(rec[dateIdx])
		closeStr := strings.TrimSpace(rec[closeIdx])
		volStr := strings.TrimSpace(rec[volumeIdx])
		if dateStr == "" || closeStr == "" {
			continue
		}
		closeVal, err := strconv.ParseFloat(closeStr, 64)
		if err != nil {
			continue
		}
		volVal := 0.0
		if volStr != "" {
			if v, err := strconv.ParseFloat(volStr, 64); err == nil {
				volVal = v
			}
		}
		ts, err := parseCSVTime(dateStr)
		if err != nil {
			continue
		}
		// Keep bid/ask = close for simplicity (CSV does not provide quotes)
		err = appender.Append(code, ts.Format("2006-01-02 15:04:05.000"), closeVal, volVal, closeVal, closeVal)
		if err != nil {
			panic(err)
		}
		loaded++
		if maxRows > 0 && loaded >= maxRows {
			break
		}
	}

	success, fail, err := appender.Close()
	if err != nil {
		return loaded, err
	}
	if fail > 0 {
		return loaded, fmt.Errorf("appender closed with %d failed rows", fail)
	}
	if success != int64(loaded) {
		return loaded, fmt.Errorf("appender closed with mismatched loaded rows: expected %d, got %d", loaded, success)
	}
	return loaded, nil
}

func parseCSVTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	layouts := []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"20060102",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format: %q", s)
}
