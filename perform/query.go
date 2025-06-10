package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type QueryScenario struct {
	Database   string // Neo HTTP address
	Table      string // Table to query
	QueryTotal int
	Time       int64
}

func (q *QueryScenario) Run() {
	ts1 := time.Now()
	defer func() {
		ts2 := time.Now()
		elapse := ts2.Sub(ts1)
		fmt.Printf("Query completed in %s (avg. %.2f queries/s)\n", elapse, float64(q.QueryTotal)/elapse.Seconds())
	}()

	for i := 0; i < q.QueryTotal; i++ {
		url := q.Database + "/db/query?q=" + url.QueryEscape(
			fmt.Sprintf(`select NAME,TIME,VALUE from %s where name = '%s' and time >= %d limit 1`,
				q.Table, "perform", q.Time+int64(i)))
		rsp, err := httpClient.Get(url)
		if err != nil {
			println("Error querying:", err.Error())
			return
		}
		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			println("Error reading response body:", err.Error())
			return
		}
		rsp.Body.Close()

		if rsp.StatusCode != http.StatusOK {
			println("Error response from server:", rsp.Status)
			return
		}

		println(string(body))
	}
}
