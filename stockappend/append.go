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
	"syscall"
	"time"

	"github.com/machbase/neo-engine/v8/native"
	"github.com/machbase/neo-server/v8/api"
	"github.com/machbase/neo-server/v8/api/machcli"
)

var host = "127.0.0.1"
var port = 5656
var user = "sys"
var password = "manager"
var appendTps = float64(1000) // 1000 TPS

// Usage: go run ./stockappend -tps <tps> -h <host> -p <port> -u <user> -P <password>
func main() {
	flag.StringVar(&host, "h", host, "server host")
	flag.IntVar(&port, "p", port, "server port")
	flag.StringVar(&user, "u", user, "user")
	flag.StringVar(&password, "P", password, "password")
	flag.Float64Var(&appendTps, "tps", appendTps, "append TPS (5 = 20ms interval)")
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

func AppendData(ctx context.Context, db *machcli.Database, tps float64) func() {
	codes := strings.Split(codesTxt, "\n")
	interval := time.Duration(float64(time.Second) / tps)
	gen := NewDataGenerator(codes, interval)

	var conn *machcli.Conn
	if c, err := db.Connect(ctx, api.WithPassword(user, password)); err != nil {
		panic(err)
	} else {
		conn = c.(*machcli.Conn)
	}

	appender, err := conn.Appender(ctx, "stock_tick")
	if err != nil {
		panic(err)
	}

	go gen.Start(func(data Data) {
		code := data.Code
		ts := data.Timestamp
		closeVal := data.Price
		volVal := data.Volume
		bidVal := data.BidPrice
		askVal := data.AskPrice
		err := appender.Append(code, ts, closeVal, volVal, bidVal, askVal)
		if err != nil {
			fmt.Printf("append error: %v\n", err)
		}
	})

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
		case tickTime := <-ticker.C:
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

				// Distribute timestamps within the tick interval
				offset := time.Duration(i) * baseInterval
				if offset > tickInterval {
					offset = tickInterval
				}
				timestamp := tickTime.Add(-tickInterval + offset)

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
