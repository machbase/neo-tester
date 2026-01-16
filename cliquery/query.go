package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machcli"
)

func main() {
	var nClient = 50
	var nCount = 1000

	db, err := machcli.NewDatabase(&machcli.Config{
		Host:         "127.0.0.1",
		Port:         5656,
		MaxOpenConn:  -1,
		MaxOpenQuery: -1,
	})
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	conn, err := db.Connect(ctx, api.WithPassword("sys", "manager"))
	if err != nil {
		panic(err)
	}
	result := conn.Exec(ctx, "CREATE TAG TABLE IF NOT EXISTS tag (name varchar(80) primary key, time DATETIME basetime, value DOUBLE)")
	if result.Err() != nil {
		panic(result.Err())
	}
	conn.Close()

	var wg sync.WaitGroup
	for i := 0; i < nClient; i++ {
		wg.Add(1)
		if i == 0 {
			go func(ctx context.Context, clientId int) {
				defer wg.Done()
				conn, err := db.Connect(ctx, api.WithPassword("sys", "manager"))
				if err != nil {
					panic(err)
				}
				for j := 0; j < nCount; j++ {
					result := conn.Exec(ctx, "INSERT INTO tag(name, time, value) VALUES('tag1', now, 123.45)")
					if result.Err() != nil {
						panic(result.Err())
					}
				}
				defer conn.Close()
			}(ctx, i)
		} else {
			go func(ctx context.Context, clientId int) {
				defer wg.Done()
				var conn *machcli.Conn
				// if c, err := db.Connect(ctx, api.WithPassword("sys", "manager")); err != nil {
				// 	panic(err)
				// } else {
				// 	conn = c.(*machcli.Conn)
				// }
				// defer conn.Close()

				for j := 0; j < nCount; j++ {
					if c, err := db.Connect(ctx, api.WithPassword("sys", "manager")); err != nil {
						panic(err)
					} else {
						conn = c.(*machcli.Conn)
					}

					tick := time.Now()
					r, err := conn.Query(ctx, "SELECT * FROM tag WHERE name='tag1' LIMIT 100, 100")
					if err != nil {
						fmt.Printf("client %d, elapsed %v\n", clientId, time.Since(tick))
						panic(err)
					}
					rows := r.(*machcli.Rows)
					n := 0
					for rows.Next() {
						if err := rows.Err(); err != nil {
							panic(err)
						}
						n++
					}
					if n != 100 {
						panic(fmt.Sprintf("invalid row count: %d", n))
					}
					err = rows.Close()
					if err != nil {
						panic(err)
					}

					if err := conn.Close(); err != nil {
						panic(err)
					}
				}
			}(ctx, i)
		}
	}
	wg.Wait()
}
