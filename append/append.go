package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// scenario-name -> sql-texts
var scenarios = map[string]func(n, i int) string{}

func init() {
	scenarios["default"] = func(workerId, round int) string {
		rn := 30
		b := &bytes.Buffer{}
		var now = time.Now().UnixNano()
		for j := 0; j < rn; j++ {
			b.WriteString(fmt.Sprintf(`{"NAME":"work-%d-%d","TIME":%d,"VALUE":%f}%s`,
				workerId, round%10000, now+int64(j), float64(j)/float64(round+1), "\n"))
		}
		b.WriteString("\n")
		return b.String()
	}
}

func main() {
	numberOfWorkers := 1
	neoHttpAddr := "http://127.0.0.1:5654"
	numberOfRuns := 1
	scenario := "default"
	useTql := false
	useCache := false

	flag.StringVar(&neoHttpAddr, "neo-http", neoHttpAddr, "Neo HTTP address")
	flag.IntVar(&numberOfWorkers, "n", numberOfWorkers, "Number of workers to use")
	flag.IntVar(&numberOfRuns, "r", numberOfRuns, "Number of runs")
	flag.StringVar(&scenario, "scenario", scenario, "Scenario to run")
	flag.BoolVar(&useTql, "tql", useTql, "Use TQL")
	flag.BoolVar(&useCache, "cache", useCache, "Use cache")
	flag.Parse()

	dataGen := scenarios[scenario]

	runChan := make(chan time.Duration, 1000)
	queryChan := make(chan time.Duration, 1000)

	stat := NewStat(numberOfWorkers, numberOfRuns)
	stat.Start(runChan, queryChan)

	wg := sync.WaitGroup{}
	for i := 0; i < numberOfWorkers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			for r := 0; r < numberOfRuns; r++ {
				start := time.Now()

				var queryElapse time.Duration
				queryElapse = appendNeoHttp(neoHttpAddr, dataGen(workerId, r))

				runElapse := time.Since(start)
				runChan <- runElapse
				queryChan <- queryElapse
			}
		}(i)
	}
	wg.Wait()
	close(runChan)
	close(queryChan)
	stat.Stop()
}

var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
	},
}

func appendNeoHttp(neoHttpAddr string, payload string) time.Duration {
	req, err := http.NewRequest("POST", neoHttpAddr+"/db/write/EXAMPLE?method=append&timeformat=ns", bytes.NewBufferString(payload))
	if err != nil {
		fmt.Println("Failed to create request:", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to select data:", err)
		os.Exit(1)
	}
	if rsp.StatusCode != http.StatusOK {
		dumpResponse(rsp, fmt.Sprint("Failed to append data:", payload))
		os.Exit(1)
	}

	content, err := io.ReadAll(rsp.Body)
	if err != nil {
		fmt.Println("Failed to read response body:", err)
		os.Exit(1)
	}
	rsp.Body.Close()

	jsonStr := string(content)
	success := gjson.Get(jsonStr, "success").Bool()
	if !success {
		reason := gjson.Get(jsonStr, "reason").String()
		fmt.Println("Failed to select data:", reason)
		os.Exit(1)
	}
	rows := gjson.Get(jsonStr, "data.rows").Array()
	_ = rows
	elapseStr := gjson.Get(jsonStr, "elapse").String()
	elapse, err := time.ParseDuration(elapseStr)
	if err != nil {
		fmt.Println("Failed to parse elapse:", err)
		os.Exit(1)
	}
	return elapse
}

func dumpResponse(rsp *http.Response, msg string) {
	fmt.Println("Log:", msg)
	fmt.Println("Status:", rsp.Status)
	fmt.Println("Header:")
	for k, v := range rsp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}
	fmt.Println("Body:")
	io.Copy(os.Stdout, rsp.Body)
}

type Stat struct {
	runCount      int64
	prevRunCount  int64
	runElapsedSum time.Duration
	runElapseMin  time.Duration
	runElapseMax  time.Duration

	queryElapsedSum time.Duration
	queryElapsedMin time.Duration
	queryElapsedMax time.Duration

	startTime time.Time
	closeCh   chan struct{}
	closeWg   sync.WaitGroup
	ticker    *time.Ticker

	workers int
	runs    int
}

func NewStat(worker, run int) *Stat {
	return &Stat{
		closeCh:   make(chan struct{}),
		ticker:    time.NewTicker(10 * time.Second),
		startTime: time.Now(),
		workers:   worker,
		runs:      run,
	}
}

func (s *Stat) Start(runCh chan time.Duration, queryCh chan time.Duration) {
	s.closeWg.Add(1)
	go func() {
		defer s.closeWg.Done()
		for {
			select {
			case d := <-runCh:
				s.runCount++
				s.runElapsedSum += d
				if s.runElapseMin == 0 || d < s.runElapseMin {
					s.runElapseMin = d
				}
				if d > s.runElapseMax {
					s.runElapseMax = d
				}
			case d := <-queryCh:
				s.queryElapsedSum += d
				if s.queryElapsedMin == 0 || d < s.queryElapsedMin {
					s.queryElapsedMin = d
				}
				if d > s.queryElapsedMax {
					s.queryElapsedMax = d
				}
			case <-s.ticker.C:
				s.Print()
			case <-s.closeCh:
				return
			}
		}
	}()
}

func (s *Stat) Stop() {
	close(s.closeCh)
	s.closeWg.Wait()
	s.Print()
}

var printer = message.NewPrinter(language.English)

func (s *Stat) Print() {
	thisRunCount := s.runCount - s.prevRunCount

	printer.Println(" Elapsed:", time.Since(s.startTime), "Workers:", s.workers, "Runs:", s.runs)
	if s.runCount == 0 {
		return
	}
	printer.Println(" Query runs:", s.runCount, "/", s.workers*s.runs, ", This cycle:", thisRunCount)
	printer.Println(" http   avg:", s.runElapsedSum/time.Duration(s.runCount), "min:", s.runElapseMin, "max:", s.runElapseMax)
	printer.Println(" query  avg:", s.queryElapsedSum/time.Duration(s.runCount), "min:", s.queryElapsedMin, "max:", s.queryElapsedMax)
	fmt.Println()

	s.prevRunCount = s.runCount
}
