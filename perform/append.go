package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/tidwall/gjson"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type AppendScenario struct {
	Database   string
	Table      string
	DataTotal  int
	DataPerReq int
}

func (cfg AppendScenario) Value(remain int) float64 {
	// Generate a random value for the record
	return rand.Float64() * 10
}

func (cfg AppendScenario) Run() {
	cfg.run(cfg.Value)
}

func (cfg AppendScenario) run(value func(int) float64) {
	ts1 := time.Now()
	defer func() {
		ts2 := time.Now()
		p := message.NewPrinter(language.English)
		elapse := ts2.Sub(ts1)
		p.Printf("Append %d records to %s in %s (avg. %.2f records/s)\n",
			cfg.DataTotal, cfg.Table, elapse, float64(cfg.DataTotal)/float64(elapse.Seconds()))
	}()

	for remain := cfg.DataTotal; remain > 0; remain -= cfg.DataPerReq {
		firstTime := time.Now().UnixNano()
		lastTime := firstTime
		data := &bytes.Buffer{}
		for i := 0; i < cfg.DataPerReq; i++ {
			lastTime += 1
			dataValue := value(remain - i)
			if i == 0 {
				data.WriteString("NAME,TIME,VALUE\n") // CSV header
			}
			rec := []byte(fmt.Sprintf("%s,%d,%.4f\n", tagName, lastTime, dataValue))
			data.Write(rec)
		}
		data.Write([]byte("\n")) // Ensure the last record is followed by a newline

		req, err := http.NewRequest("POST", cfg.Database+"/db/write/"+cfg.Table+"?method=append&timeformat=ns&header=columns", data)
		if err != nil {
			fmt.Println("Error creating request:", err.Error())
			return
		}
		req.Header.Set("Content-Type", "text/csv")

		rsp, err := httpClient.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err.Error())
			return
		}
		if rsp.StatusCode != http.StatusOK {
			fmt.Println("Error response from server:", rsp.Status)
			return
		}
		defer rsp.Body.Close()
		content, err := io.ReadAll(rsp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err.Error())
			return
		}
		success := gjson.Get(string(content), "success").Bool()
		tf := time.Now().Format("2006/01/02 15:04:05.999")
		if success {
			fmt.Println(tf, "Append done:", firstTime, lastTime, string(content))
		} else {
			fmt.Println(tf, "Append failed:", string(content))
		}
	}
}
