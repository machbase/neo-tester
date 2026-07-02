package main

import (
	"context"
	"fmt"

	"github.com/machbase/neo-client/api"
	"github.com/machbase/neo-client/machgo"
)

func main() {
	conf := &machgo.Config{
		Host: "127.0.0.1", // Machbase 서버 호스트
		Port: 5656,        // Machbase 네이티브 포트
	}

	// 데이터베이스 인스턴스 생성
	// API 사용 방식은 machcli와 동일
	mdb, err := machgo.NewDatabase(conf)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Connection A: statement 재사용을 적극적으로 사용
	connA, err := mdb.Connect(
		ctx,
		api.WithPassword("sys", "manager"),
		api.WithStatementCache(api.StatementCacheAuto),
	)
	if err != nil {
		panic(err)
	}
	defer connA.Close()

	// Connection B: 이 연결에서만 statement 재사용 비활성화
	connB, err := mdb.Connect(
		ctx,
		api.WithPassword("sys", "manager"),
		api.WithStatementCache(api.StatementCacheOff),
	)
	if err != nil {
		panic(err)
	}
	defer connB.Close()

	// A: create table
	sqlCreateTable := "create tag table if not exists stmtcache (name varchar(80) primary key, time datetime basetime, value double)"
	result := connA.Exec(ctx, sqlCreateTable)
	if err := result.Err(); err != nil {
		panic(err)
	}

	// A: insert data, statement cache 활성화 상태에서 insert 수행
	sqlInsert := "insert into stmtcache values (?, ?, ?)"
	result = connA.Exec(ctx, sqlInsert, "Alice", "2024-06-01 00:00:00", 123.45)
	if err := result.Err(); err != nil {
		panic(err)
	}

	// B: drop table
	result = connB.Exec(ctx, "drop table stmtcache")
	if err := result.Err(); err != nil {
		panic(err)
	}

	// B: re-create table from a connection other than A
	result = connB.Exec(ctx, sqlCreateTable)
	if err := result.Err(); err != nil {
		panic(err)
	}

	// A: insert data again, statement cache 활성화 상태에서 insert 수행
	result = connA.Exec(ctx, sqlInsert, "Bob", "2024-06-02 00:00:00", 678.90)
	if err := result.Err(); err != nil {
		panic(fmt.Sprintf("Issue: machbase/neo#1395, %v", err))
	}
}
