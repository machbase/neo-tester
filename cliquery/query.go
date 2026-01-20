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

	"github.com/machbase/neo-engine/v8/native"
	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machcli"
)

func main() {
	var nClient = 50
	var nCount = 1000
	var nFetch = 100
	var doCpuProfile = false
	var doOSThreadLock = false
	var doCreateData = false
	var doPreparedStmt = false
	var sessionElapsed []time.Duration
	var host = "127.0.0.1"
	var port = 5656
	var user = "sys"
	var password = "manager"

	flag.IntVar(&nClient, "c", nClient, "number of clients")
	flag.IntVar(&nCount, "n", nCount, "number of queries per client")
	flag.IntVar(&nFetch, "f", nFetch, "number of rows to fetch per query")
	flag.StringVar(&host, "h", host, "server host")
	flag.IntVar(&port, "p", port, "server port")
	flag.StringVar(&user, "u", user, "user")
	flag.StringVar(&password, "P", password, "password")
	flag.BoolVar(&doOSThreadLock, "T", doOSThreadLock, "enable OS thread lock")
	flag.BoolVar(&doCpuProfile, "prof", doCpuProfile, "enable cpu profiling")
	flag.BoolVar(&doCreateData, "create", doCreateData, "create initial data")
	flag.BoolVar(&doPreparedStmt, "prep", doPreparedStmt, "use prepared statement")
	flag.Parse()

	fmt.Println("Neo Engine Version:", native.Version, "Build:", native.GitHash)
	var start = time.Now()
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
	if doCreateData {
		result := conn.Exec(ctx, "CREATE TAG TABLE IF NOT EXISTS tag (name varchar(80) primary key, time DATETIME basetime, value DOUBLE)")
		if result.Err() != nil {
			panic(result.Err())
		}

		for j := 0; j < nFetch*2; j++ {
			result := conn.Exec(ctx, "INSERT INTO tag(name, time, value) VALUES('tag1', now, 123.45)")
			if result.Err() != nil {
				panic(result.Err())
			}
		}
	}
	conn.Close()

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

	for i := 0; i < nClient; i++ {
		wg.Add(1)

		go func(ctx context.Context, clientId int) {
			defer wg.Done()
			if doOSThreadLock {
				runtime.LockOSThread()
			}
			<-startCh
			var conn *machcli.Conn

			if c, err := db.Connect(ctx, api.WithPassword("sys", "manager")); err != nil {
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
			if doPreparedStmt {
				RunPreparedQuery(ctx, clientId, conn, nCount, nFetch)
			} else {
				RunQuery(ctx, clientId, conn, nCount, nFetch)
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

func RunQuery(ctx context.Context, clientId int, conn api.Conn, nCount int, nFetch int) {
	for j := 0; j < nCount; j++ {
		tick := time.Now()
		r, err := conn.Query(ctx, "SELECT * FROM tag WHERE name='tag1' LIMIT ?", nFetch)
		if err != nil {
			fmt.Printf("Query error, client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
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
			var v float64
			if err := rows.Scan(&name, &t, &v); err != nil {
				panic(err)
			}
			if name != "tag1" {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		if n != nFetch {
			panic(fmt.Sprintf("invalid row count: %d", n))
		}
		tick = time.Now()
		err = rows.Close()
		if err != nil {
			fmt.Printf("Close error, client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
	}
}

func RunPreparedQuery(ctx context.Context, clientId int, conn api.Conn, nCount int, nFetch int) {
	var stmt *machcli.PreparedStmt
	if s, err := conn.Prepare(ctx, "SELECT * FROM tag WHERE name='tag1' LIMIT ?"); err != nil {
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
		r, err := stmt.Query(ctx, nFetch)
		if err != nil {
			fmt.Printf("Query error, client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
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
			var v float64
			if err := rows.Scan(&name, &t, &v); err != nil {
				panic(err)
			}
			if name != "tag1" {
				panic(fmt.Sprintf("invalid name: %s", name))
			}
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		if n != nFetch {
			panic(fmt.Sprintf("invalid row count: %d repeat: %d", n, j))
		}
		tick = time.Now()
		if err = rows.Close(); err != nil {
			fmt.Printf("Close error, client %d, elapsed %v %s\n", clientId, time.Since(tick), err.Error())
			return
		}
	}
}
