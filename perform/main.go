package main

import (
	"flag"
	"net/http"
	"os"
	"time"
)

func main() {
	scenario := ""
	database := "http://127.0.0.1:5654"
	table := "tag"
	queryTime := int64(0)

	flag.Usage = Usage
	flag.StringVar(&scenario, "scenario", scenario, "Scenario to run (append, query, lsl,usl)")
	flag.StringVar(&database, "database", database, "Neo HTTP address")
	flag.StringVar(&table, "table", table, "Table to use")
	flag.Int64Var(&queryTime, "time", int64(queryTime), "Query time in nanoseconds (for query scenario)")
	flag.Parse()
	if len(os.Args) < 2 {
		println("Usage: perform <command> [flags]")
		return
	}
	switch scenario {
	case "append":
		app := &AppendScenario{
			Database:   database,
			Table:      table,
			DataTotal:  1000_000,
			DataPerReq: 1000,
		}
		app.Run()
	case "query":
		app := &QueryScenario{
			Database:   database,
			Table:      table,
			QueryTotal: 500,
			Time:       queryTime,
		}
		app.Run()
	case "lsl":
		app := &LSLScenario{
			AppendScenario: AppendScenario{
				Database:   database,
				Table:      table,
				DataTotal:  100,
				DataPerReq: 100,
			},
			LSL: -1.23,
		}
		app.Run()
	case "usl":
		app := &LSLScenario{
			AppendScenario: AppendScenario{
				Database:   database,
				Table:      table,
				DataTotal:  100,
				DataPerReq: 100,
			},
			LSL: 100.5,
		}
		app.Run()
	default:
		println("Unknown command:", scenario)
		Usage()
	}
}

func Usage() {
	println("Usage: perform [flags]")
	println("flags:")
	flag.PrintDefaults()
}

const tagName = "perform"

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
	},
	Timeout: 30 * time.Second,
}
