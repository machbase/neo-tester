package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/machbase/neo-client/api"
	"github.com/machbase/neo-client/machgo"
	"github.com/machbase/neo-server/v8/jsh/lib/pretty"
)

var host = "127.0.0.1"
var port = 5656
var user = "sys"
var password = "manager"
var createTables = false
var appendTps = float64(1000) // 1000 TPS

// Usage: go run ./stockappend -tps <tps> -h <host> -p <port> -u <user> -P <password>
func main() {
	flag.StringVar(&host, "h", host, "server host")
	flag.IntVar(&port, "p", port, "server port")
	flag.StringVar(&user, "u", user, "user")
	flag.StringVar(&password, "P", password, "password")
	flag.Float64Var(&appendTps, "tps", appendTps, "append TPS (5 = 20ms interval)")
	flag.BoolVar(&createTables, "create", false, "create tables and rollups")
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

	// create tables if not exists
	if createTables {
		CreateTablesIfNotExists(ctx, db)
	}

	// start appending data
	if appendTps > 0 {
		stopFunc := AppendData(ctx, db, appendTps)
		defer stopFunc()
	}
	interruptSignal := make(chan os.Signal, 1)
	signal.Notify(interruptSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Press Ctrl+C to stop...")
	<-interruptSignal
	fmt.Println("Stopping...")
}

//go:embed stock_codes.txt
var codesTxt string

func AppendData(ctx context.Context, db *machgo.Database, tps float64) func() {
	codes := strings.Split(codesTxt, "\n")
	interval := time.Duration(float64(time.Second) / tps)
	gen := NewDataGenerator(codes, interval)

	var conn *machgo.Conn
	if c, err := db.Connect(ctx, api.WithPassword(user, password), api.WithIOMetrics(true)); err != nil {
		panic(err)
	} else {
		conn = c.(*machgo.Conn)
	}

	appender, err := conn.Appender(ctx, "stock_tick")
	if err != nil {
		panic(err)
	}

	count := uint64(0)
	go gen.Start(func(data Data) {
		code := data.Code
		ts := data.Timestamp
		closeVal := data.Price
		volVal := data.Volume
		bidVal := data.BidPrice
		askVal := data.AskPrice
		err := appender.Append(code, ts, closeVal, volVal, bidVal, askVal)
		if err != nil {
			panic(err)
		}
		atomic.AddUint64(&count, 1)
	})

	go func() {
		statTicker := time.NewTicker(1 * time.Second)
		defer statTicker.Stop()
		var statConn *machgo.Conn
		if c, err := db.Connect(ctx, api.WithPassword(user, password)); err != nil {
			panic(err)
		} else {
			statConn = c.(*machgo.Conn)
			defer statConn.Close()
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		tick := time.Now()
		lastCount := uint64(0)
		for {
			select {
			case <-gen.Done():
				return
			case <-statTicker.C:
				stats, err := ShowRollupGap(ctx, statConn)
				if err != nil {
					fmt.Println("Error querying rollup elapsed time:", err)
					continue
				}
				for _, stat := range stats {
					fmt.Printf("%s Rollup %s elapsed: %s, gap: %s\n",
						time.Now().Format("2006-01-02 15:04:05"),
						stat.RollupName, pretty.Durations(stat.Elapsed), pretty.Ints(stat.Gap))
				}
			case now := <-ticker.C:
				elapsed := now.Sub(tick).Seconds()
				readBytes, writeBytes, _ := conn.ResetIOMetrics()
				tick = now
				cnt := atomic.LoadUint64(&count)
				tps := float64(cnt-lastCount) / elapsed
				lastCount = cnt
				writeBytesPerSec := float64(writeBytes) / elapsed
				readBytesPerSec := float64(readBytes) / elapsed
				fmt.Printf("%s TPS: %s/s Read: %s (%s/s), Write: %s (%s/s)\n",
					now.Format("2006-01-02 15:04:05"),
					pretty.Ints(tps),
					pretty.Bytes(readBytes), pretty.Bytes(readBytesPerSec),
					pretty.Bytes(writeBytes), pretty.Bytes(writeBytesPerSec))
			}
		}
	}()

	// return stop function
	return func() {
		gen.Stop()
		appender.Close()
		conn.Close()
	}
}

type Data struct {
	Timestamp time.Time
	Code      string
	Price     float64
	Volume    float64
	BidPrice  float64
	AskPrice  float64
}

// DataGenerator simulates real-time stock data generation for the given stock codes at specified intervals.
// the generating interval should be randomized within `Interval+-(Interval/4)` to better simulate real-time data.
// The generated data is sent to the provided Callback function.
// The generated data code is randomly selected from the Codes slice.
//
// Usage:
//
//	dg := NewDataGenerator([]string{"AAPL", "GOOG", "MSFT"}, time.Microsecond*500)
//	go dg.Start(func(data Data) {
//	    fmt.Println(data)
//	})
//	// To run the generator for a specific duration:
//	time.Sleep(10 * time.Second)
//	// To stop the generator:
//	dg.Stop()
type DataGenerator struct {
	Codes    []string
	Interval time.Duration

	stopChan chan struct{}
	stopOnce sync.Once
}

func NewDataGenerator(codes []string, interval time.Duration) *DataGenerator {
	return &DataGenerator{
		Codes:    codes,
		Interval: interval,
		stopChan: make(chan struct{}),
	}
}
func (dg *DataGenerator) Start(callback func(Data)) {
	if callback == nil || len(dg.Codes) == 0 {
		return
	}
	baseInterval := dg.Interval
	if baseInterval <= 0 {
		baseInterval = time.Second
	}

	// Use fixed tick interval for timer efficiency
	tickInterval := 100 * time.Millisecond
	if baseInterval > tickInterval {
		tickInterval = baseInterval
	}

	type codeState struct {
		price      float64
		anchor     float64
		baseVolume float64
		volatility float64
		drift      float64
	}
	stateByCode := make(map[string]*codeState, len(dg.Codes))
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, code := range dg.Codes {
		basePrice := 50 + rnd.Float64()*150
		stateByCode[code] = &codeState{
			price:      basePrice,
			anchor:     basePrice,
			baseVolume: 100 + rnd.Float64()*9000,
			volatility: 0.001 + rnd.Float64()*0.01,
			drift:      (rnd.Float64() - 0.5) * 0.0002,
		}
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dg.stopChan:
			return
		case timeTick := <-ticker.C:
			// Calculate how many data points to generate in this tick
			count := int(tickInterval / baseInterval)
			remainder := float64(tickInterval%baseInterval) / float64(baseInterval)
			if rnd.Float64() < remainder {
				count++
			}
			if count == 0 {
				count = 1
			}

			// Generate batch of data points
			for i := 0; i < count; i++ {
				code := dg.Codes[rnd.Intn(len(dg.Codes))]
				state := stateByCode[code]
				if state == nil {
					basePrice := 50 + rnd.Float64()*150
					state = &codeState{
						price:      basePrice,
						anchor:     basePrice,
						baseVolume: 100 + rnd.Float64()*9000,
						volatility: 0.001 + rnd.Float64()*0.01,
						drift:      (rnd.Float64() - 0.5) * 0.0002,
					}
					stateByCode[code] = state
				}

				interval := randomizedInterval(rnd, baseInterval)
				dt := float64(interval) / float64(time.Second)
				if dt <= 0 {
					dt = 1
				}
				shock := rnd.NormFloat64() * state.volatility * math.Sqrt(dt)
				move := shock + state.drift*dt
				price := state.price * (1 + move)
				price += (state.anchor - price) * 0.001
				if price < 1 {
					price = 1
				}
				state.price = price

				absMove := math.Abs(move)
				volume := state.baseVolume * (1 + absMove*50 + rnd.Float64()*0.2)
				if volume < 1 {
					volume = 1
				}
				spreadPct := 0.0005 + rnd.Float64()*0.0015
				spread := price * spreadPct
				bid := price - spread/2
				if bid < 0.01 {
					bid = 0.01
				}
				ask := price + spread/2

				timestamp := timeTick.Add(time.Duration(i))

				callback(Data{
					Timestamp: timestamp,
					Code:      code,
					Price:     price,
					Volume:    volume,
					BidPrice:  bid,
					AskPrice:  ask,
				})
			}
		}
	}
}

func (dg *DataGenerator) Stop() {
	dg.stopOnce.Do(func() {
		close(dg.stopChan)
	})
}

func (dg *DataGenerator) Done() <-chan struct{} {
	return dg.stopChan
}

func randomizedInterval(rnd *rand.Rand, base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	jitter := base / 4
	if jitter == 0 {
		return base
	}
	min := -jitter
	max := jitter
	span := int64(max-min) + 1
	return base + time.Duration(rnd.Int63n(span)+int64(min))
}

func CreateTablesIfNotExists(ctx context.Context, db api.Database) {
	conn, err := db.Connect(ctx, api.WithPassword(user, password))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	result := conn.Exec(ctx, `create tag table if not exists stock_tick (
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
	result = conn.Exec(ctx, `create tag table if not exists stock_rollup_1s (
		code       varchar(20) primary key,
		time       datetime basetime,
		sum_price  double,
		sum_volume double,
		sum_bid    double,
		sum_ask    double,
		cnt        integer,
		open       double,
		open_time  datetime,
		close      double,
		close_time datetime,
		high       double,
		low        double
	)`)
	if result.Err() != nil {
		panic(result.Err())
	}
	result = conn.Exec(ctx, `create tag table if not exists stock_rollup_1m (
    	code       varchar(20) primary key,
		time       datetime basetime,
		sum_price  double,
		sum_volume double,
		sum_bid    double,
		sum_ask    double,
		cnt        integer,
		open       double,
		open_time  datetime,
		close      double,
		close_time datetime,
		high       double,
		low        double
	)`)
	if result.Err() != nil {
		panic(result.Err())
	}
	result = conn.Exec(ctx, `create tag table if not exists stock_rollup_1h (
		code       varchar(20) primary key,
		time       datetime basetime,
		sum_price  double,
		sum_volume double,
		sum_bid    double,
		sum_ask    double,
		cnt        integer,
		open       double,
		open_time  datetime,
		close      double,
		close_time datetime,
		high       double,
		low        double
	)`)
	if result.Err() != nil {
		panic(result.Err())
	}

	result = conn.Exec(ctx, `create rollup rollup_stock_1s
		into (stock_rollup_1s)
		as (
			select
				code,
				date_trunc('second', time) as time,
				sum(price) as sum_price,
				sum(volume) as sum_volume,
				sum(bid_price) as sum_bid,
				sum(ask_price) as sum_ask,
				count(*) as cnt,
				first(time, price) as open,
				min(time) as open_time,
				last(time, price) as close,
				max(time) as close_time,
				max(price) as high,
				min(price) as low
			from stock_tick
			group by code, time
		)
		interval 1 sec`)
	if result.Err() != nil {
		panic(result.Err())
	}
	result = conn.Exec(ctx, `create rollup rollup_stock_1m
		into (stock_rollup_1m)
		as (
			select
				code,
				date_trunc('minute', time) as time,
				sum(sum_price) as sum_price,
				sum(sum_volume) as sum_volume,
				sum(sum_bid) as sum_bid,
				sum(sum_ask) as sum_ask,
				sum(cnt) as cnt,
				first(open_time, open) as open,
				min(open_time) as open_time,
				last(close_time, close) as close,
				max(close_time) as close_time,
				max(high) as high,
				min(low) as low
			from stock_rollup_1s
			group by code, time
		)
		interval 1 min`)
	if result.Err() != nil {
		panic(result.Err())
	}
	result = conn.Exec(ctx, `create rollup rollup_stock_1h
		into (stock_rollup_1h)
		as (
			select
				code,
				date_trunc('hour', time) as time,
				sum(sum_price) as sum_price,
				sum(sum_volume) as sum_volume,
				sum(sum_bid) as sum_bid,
				sum(sum_ask) as sum_ask,
				sum(cnt) as cnt,
				first(open_time, open) as open,
				min(open_time) as open_time,
				last(close_time, close) as close,
				max(close_time) as close_time,
				max(high) as high,
				min(low) as low
			from stock_rollup_1m
			group by code, time
		)
		interval 1 hour`)
	if result.Err() != nil {
		panic(result.Err())
	}
}

type RollupStat struct {
	RollupName string
	Elapsed    time.Duration
	Gap        uint64
}

func ShowRollupGap(ctx context.Context, statConn *machgo.Conn) ([]RollupStat, error) {
	var ret []RollupStat
	rows, err := statConn.Query(ctx, `
		select
			C.rollup_name, 
			C.last_elapsed_msec as elapsed_msec,
			B.table_end_rid - C.end_rid as gap
		from
			m$sys_tables A,
			v$storage_tag_tables B,
			v$rollup C
		where
			C.SOURCE_TABLE = A.NAME
		and B.ID = A.ID
		order by C.rollup_name
		`)
	if err != nil {
		return ret, err
	}
	defer rows.Close()
	var curStat RollupStat
	for rows.Next() {
		var rollupName string
		var elapsedMsec float64
		var gap uint64
		if err := rows.Scan(&rollupName, &elapsedMsec, &gap); err != nil {
			fmt.Println("Error scanning rollup elapsed time:", err)
			continue
		}

		if rollupName != curStat.RollupName {
			if curStat.RollupName != "" {
				ret = append(ret, curStat)
			}
			curStat = RollupStat{RollupName: rollupName}
		}
		var elapsed = time.Duration(elapsedMsec * float64(time.Millisecond))
		if elapsed > curStat.Elapsed {
			curStat.Elapsed = elapsed
		}
		if gap > curStat.Gap {
			curStat.Gap = gap
		}
	}
	if curStat.RollupName != "" {
		ret = append(ret, curStat)
	}
	return ret, nil
}
