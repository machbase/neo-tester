// droptrace executes "DROP TABLE <table> CASCADE" and traces whether the
// server returns an error and whether the table and its rollups are removed.
//
// Run it while ./stockappend is appending data to reproduce the
// "Resource busy" case (https://github.com/machbase/dbms-nfx/issues/4344).
//
// Usage:
//
//	go run ./droptrace -table stock_tick -via query
//	go run ./droptrace -table sys.stock_tick -via exec
//	go run ./droptrace -table stock_tick -via shell -neo ../neo-server/tmp/machbase-neo
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	client "github.com/machbase/neo-client/v2"
)

var (
	host      = "127.0.0.1"
	port      = 5656
	httpPort  = 5654
	user      = "sys"
	password  = "manager"
	tableName = "stock_tick"
	via       = "query"
	neoBin    = "../neo-server/tmp/machbase-neo"
	repeat    = 1
	interval  = time.Second
)

func main() {
	flag.StringVar(&host, "h", host, "server host")
	flag.IntVar(&port, "p", port, "server port")
	flag.IntVar(&httpPort, "http-port", httpPort, "server http port (for -via shell)")
	flag.StringVar(&user, "u", user, "user")
	flag.StringVar(&password, "P", password, "password")
	flag.StringVar(&tableName, "table", tableName, "table name to drop, e.g. stock_tick or sys.stock_tick")
	flag.StringVar(&via, "via", via, "execution path: exec (Conn.ExecContext), query (Conn.QueryContext, same as jsh machcli), shell (machbase-neo shell sql)")
	flag.StringVar(&neoBin, "neo", neoBin, "machbase-neo binary path (for -via shell)")
	flag.IntVar(&repeat, "n", repeat, "number of attempts, stops on the first success")
	flag.DurationVar(&interval, "interval", interval, "interval between attempts")
	flag.Parse()

	ctx := context.Background()
	dsn := fmt.Sprintf("user=%s;password=%s;server=%s:%d", user, password, host, port)
	db, err := sql.Open("machbase", dsn)
	if err != nil {
		fatal("open", err)
	}
	defer db.Close()

	owner, baseName, err := parseTableName(tableName, user)
	if err != nil {
		fatal("table", err)
	}
	sqlText := fmt.Sprintf("DROP TABLE %s CASCADE", tableName)

	for i := 1; i <= repeat; i++ {
		fmt.Printf("==== attempt %d/%d via=%s: %s\n", i, repeat, via, sqlText)
		before := snapshot(ctx, db, owner, baseName)
		before.print("before")
		if before.err != nil {
			fatal("before snapshot", before.err)
		}

		tick := time.Now()
		var res outcome
		switch via {
		case "exec":
			res = dropByExec(ctx, db, sqlText)
		case "query":
			res = dropByQuery(ctx, db, sqlText)
		case "shell":
			res = dropByShell(sqlText)
		default:
			fatal("via", fmt.Errorf("unknown -via %q", via))
		}
		res.elapsed = time.Since(tick)
		res.print()

		after := snapshot(ctx, db, owner, baseName)
		after.print("after ")
		if after.err != nil {
			fatal("after snapshot", after.err)
		}
		verdict(res, before, after)

		if res.err == nil || i == repeat {
			break
		}
		time.Sleep(interval)
	}
}

type outcome struct {
	err          error
	rowsAffected int64
	message      string
	fetchable    bool
	output       string
	exitCode     int
	elapsed      time.Duration
}

func (o outcome) print() {
	fmt.Printf("  result  : elapsed=%s err=%v\n", o.elapsed.Round(time.Millisecond), o.err)
	if via == "shell" {
		fmt.Printf("  exitCode: %d\n", o.exitCode)
		fmt.Printf("  output  : %q\n", strings.TrimSpace(o.output))
		return
	}
	fmt.Printf("  message : %q rowsAffected=%d fetchable=%v\n", o.message, o.rowsAffected, o.fetchable)
}

func dropByExec(ctx context.Context, db *sql.DB, sqlText string) outcome {
	meta := &client.Meta{}
	ctx = context.WithValue(ctx, client.MetaKey, meta)
	conn, err := db.Conn(ctx)
	if err != nil {
		return outcome{err: err}
	}
	defer conn.Close()
	result, err := conn.ExecContext(ctx, sqlText)
	if err != nil {
		return outcome{err: err}
	}
	ret := outcome{message: meta.Message()}
	ret.rowsAffected, _ = result.RowsAffected()
	return ret
}

// dropByQuery follows the same path as jsh "sql" command (machcli conn.query()).
func dropByQuery(ctx context.Context, db *sql.DB, sqlText string) outcome {
	meta := &client.Meta{}
	ctx = context.WithValue(ctx, client.MetaKey, meta)
	conn, err := db.Conn(ctx)
	if err != nil {
		return outcome{err: err}
	}
	defer conn.Close()
	rows, err := conn.QueryContext(ctx, sqlText)
	if err != nil {
		return outcome{err: err}
	}
	defer rows.Close()
	ret := outcome{message: meta.Message(), fetchable: meta.IsFetchable()}
	if ret.fetchable {
		for rows.Next() {
		}
	}
	ret.err = rows.Err()
	return ret
}

// dropByShell runs the same command line as "machbase-neo shell sql <sql>".
func dropByShell(sqlText string) outcome {
	cmd := exec.Command(neoBin, "shell", "sql", sqlText)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("NEOSHELL_HOST=%s:%d", host, httpPort),
		"NEOSHELL_USER="+user,
		"NEOSHELL_PASSWORD="+password,
	)
	out, err := cmd.CombinedOutput()
	ret := outcome{output: string(out), err: err}
	if cmd.ProcessState != nil {
		ret.exitCode = cmd.ProcessState.ExitCode()
	}
	return ret
}

type state struct {
	tableExists bool
	rowCount    int64
	rollups     []string
	err         error
}

func (s state) print(label string) {
	if s.err != nil {
		fmt.Printf("  %s: error %v\n", label, s.err)
		return
	}
	fmt.Printf("  %s: table=%v rows=%d rollups(%d)=%v\n", label, s.tableExists, s.rowCount, len(s.rollups), s.rollups)
}

func parseTableName(table, defaultOwner string) (string, string, error) {
	parts := strings.Split(strings.ToUpper(table), ".")
	if len(parts) == 1 {
		parts = []string{strings.ToUpper(defaultOwner), parts[0]}
	}
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected TABLE or OWNER.TABLE: %q", table)
	}
	for _, part := range parts {
		if part == "" {
			return "", "", fmt.Errorf("empty identifier in %q", table)
		}
		for i, ch := range part {
			if !(ch >= 'A' && ch <= 'Z' || ch == '_' || i > 0 && ch >= '0' && ch <= '9') {
				return "", "", fmt.Errorf("unsupported identifier %q", part)
			}
		}
	}
	return parts[0], parts[1], nil
}

func snapshot(ctx context.Context, db *sql.DB, owner, name string) state {
	ret := state{}
	var cnt int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM M$SYS_TABLES T, M$SYS_USERS U WHERE T.USER_ID = U.USER_ID AND U.NAME = ? AND T.NAME = ?`, owner, name).Scan(&cnt); err != nil {
		ret.err = err
		return ret
	}
	ret.tableExists = cnt > 0
	if ret.tableExists {
		if err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT count(*) FROM %s.%s`, owner, name)).Scan(&ret.rowCount); err != nil {
			ret.err = err
			return ret
		}
	}
	// v$rollup has a row per source data partition, so collect distinct names.
	rows, err := db.QueryContext(ctx, `SELECT R.ROLLUP_NAME, R.SOURCE_TABLE FROM V$ROLLUP R, M$SYS_USERS U WHERE R.USER_ID = U.USER_ID AND U.NAME = ?`, owner)
	if err != nil {
		ret.err = err
		return ret
	}
	defer rows.Close()
	seen := map[string]bool{}
	prefix := "_" + name + "_DATA_"
	for rows.Next() {
		var r, src string
		if err := rows.Scan(&r, &src); err != nil {
			ret.err = err
			return ret
		}
		if !strings.HasPrefix(src, prefix) || seen[r] {
			continue
		}
		seen[r] = true
		ret.rollups = append(ret.rollups, r)
	}
	ret.err = rows.Err()
	return ret
}

func verdict(res outcome, before, after state) {
	dropped := before.tableExists && !after.tableExists
	switch {
	case res.err == nil && !dropped:
		fmt.Println("  VERDICT : !!! client reported SUCCESS but the table still exists")
	case res.err != nil && dropped:
		fmt.Println("  VERDICT : !!! client reported ERROR but the table was dropped")
	case res.err != nil && len(after.rollups) < len(before.rollups):
		fmt.Printf("  VERDICT : client reported ERROR, table remains, but rollups changed %d -> %d\n", len(before.rollups), len(after.rollups))
	case res.err != nil:
		fmt.Println("  VERDICT : client reported ERROR, nothing changed")
	default:
		fmt.Println("  VERDICT : dropped successfully")
	}
}

func fatal(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}
