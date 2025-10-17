package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	serverAddr := "http://127.0.0.1:5654/db/write/hdcar?method=append"
	isCanData := false
	inputFile := ""
	startTime := "2006-01-02 15:04:05"
	timeColName := "t"
	offset := 0
	timeUnit := "ns"
	headerLines := 1
	headerCombine := false

	flag.StringVar(&serverAddr, "server", serverAddr, "Server address")
	flag.StringVar(&inputFile, "in", inputFile, "Input file")
	flag.IntVar(&offset, "offset", offset, "Input file line offset")
	flag.StringVar(&startTime, "start-time", startTime, "Trip start time (format: 2006-01-02 15:04:05), if not CAN data")
	flag.BoolVar(&isCanData, "can", isCanData, "CAN data")
	flag.StringVar(&timeUnit, "time-unit", timeUnit, "Timestamp unit for CAN data: s, ms, us, ns")
	flag.StringVar(&timeColName, "time-col", timeColName, "timestamp column name (t, timestamps)")
	flag.IntVar(&headerLines, "header", headerLines, "Number of header lines to read")
	flag.BoolVar(&headerCombine, "header-combine", headerCombine, "Combine all header lines with '_' (default: use first line only)")
	flag.Parse()

	if len(os.Args) < 2 {
		flag.Usage()
		return
	}

	fmt.Println("Reading file", inputFile)
	data, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Error reading file", err)
		return
	}
	defer data.Close()

	// Decide the trip ID from the file name
	var tripId = strings.TrimSuffix(strings.ToUpper(filepath.Base(inputFile)), ".CSV")
	fmt.Printf("ID: %s\n", tripId)

	// Skip to the offset line
	reader := bufio.NewReader(data)
	for i := 0; i < offset; i++ {
		_, err := reader.ReadBytes('\n')
		if err != nil {
			return // EOF
		} else {
			offset--
		}
	}

	// parse start time
	var tripStartTime time.Time
	if !isCanData {
		tripStartTime, err = time.Parse("2006-01-02 15:04:05", startTime)
		if err != nil {
			fmt.Println("Error parsing start time", err)
			return
		}

		fmt.Printf("start time: %s\n\n", tripStartTime)
	}

	// parse header line(s)
	csvReader := csv.NewReader(reader)
	var headerTrimRegex = regexp.MustCompile(`\s*\[.*\]$`)
	var headers = []string{}
	for i := range headerLines {
		fields, err := csvReader.Read()
		if err != nil {
			fmt.Println("Error reading header line", i+1, err)
			return
		}

		if i == 0 {
			headers = make([]string, len(fields))
		} else if !headerCombine {
			break
		}

		for idx, h := range fields {
			h = strings.TrimSpace(h)
			if h == "" { // all lines contains an empty field at the end
				continue
			}

			name := headerTrimRegex.ReplaceAllString(h, "")
			if i == 0 {
				headers[idx] = name
			} else {
				headers[idx] = fmt.Sprintf("%s_%s", headers[idx], name)
			}
		}
	}

	// parse body lines
	recordCount := 0
	buff := &bytes.Buffer{}
	for {
		fields, err := csvReader.Read()
		if err != nil {
			break
		}
		rec := NewRecord(headers, fields)
		var timestamp int64
		var value = 0.0
		if !isCanData { // CN7, RG3
			// TIME
			timestamp = tripStartTime.UnixNano() + int64(rec[timeColName].(float64)*1000000000) // "t"
			// VALUE
			if v, ok := rec["Speed_Kmh"]; ok {
				value = v.(float64)
			}
		} else { // CAN - raw & interpolated
			// TIME
			timestamp = int64(rec[timeColName].(float64)) // "timestamps"
			switch timeUnit {
			case "s":
				timestamp = timestamp * 1_000_000_000
			case "us":
				timestamp = timestamp * 1_000_000
			case "ms":
				timestamp = timestamp * 1_000
			}

			// VALUE
			if v, ok := rec["WHL_SpdFLVal"]; ok {
				value = v.(float64)
			}
		}

		// DATA
		jsonData, err := json.Marshal(rec)
		if err != nil {
			fmt.Println("Error marshalling record", err)
			return
		}
		escaped := strings.ReplaceAll(string(jsonData), `"`, `""`)
		//buff.Write([]byte(fmt.Sprintf("%s,%d,%f,\"%s\"\n", tripId, timestamp, value, escaped)))
		buff.Write([]byte(fmt.Sprintf("%s,%d,%f,\"%s\",\"\",\"\",\"\",\"\",\"\",\"\",0,0,0,0\n", tripId, timestamp, value, escaped)))

		recordCount++
		// send POST request for every 1000 records (lines)
		if recordCount > 1000 {
			sendHttp(serverAddr, buff)
			recordCount = 0
			buff.Reset()
			fmt.Print(".")
		}
	}

	if buff.Len() > 0 {
		sendHttp(serverAddr, buff)
	}
}

type Record map[string]any

func NewRecord(headers, fields []string) Record {
	r := Record{}
	for idx, h := range headers {
		if h == "" { // all lines contains empty string at the end
			continue
		}
		if v := fields[idx]; len(v) > 0 {
			r[h], _ = strconv.ParseFloat(v, 64)
		}
	}
	return r
}

var client = http.Client{}

func sendHttp(addr string, data io.Reader) {
	req, err := http.NewRequest("POST", addr, data)
	if err != nil {
		fmt.Println("Error creating request", err)
		return
	}
	req.Header.Set("Content-Type", "text/csv")

	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request", err)
		return
	}
	defer rsp.Body.Close()
}
