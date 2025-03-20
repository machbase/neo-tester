package main

import (
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
	flag.StringVar(&serverAddr, "server", serverAddr, "Server address")
	flag.StringVar(&inputFile, "in", inputFile, "Input file")
	flag.BoolVar(&isCanData, "can", isCanData, "CAN data")
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
	var tripStartTime time.Time

	buff := &bytes.Buffer{}
	if !isCanData {
		// read the first line which contains the trip start time
		// ex) Date:06.04.2023 Time:06:57:39
		for {
			var char [1]byte
			if n, err := data.Read(char[:]); err != nil || n == 0 {
				return // EOF
			}
			if char[0] == '\n' {
				break
			} else {
				buff.WriteByte(char[0])
			}
		}
		// fmt.Println("The first line:", buff.String())

		// parse start time
		var startTimeRegex = regexp.MustCompile(`Date:(\d{2})\.(\d{2})\.(\d{4}) Time:(\d{2}):(\d{2}):(\d{2})`)
		if match := startTimeRegex.FindStringSubmatch(buff.String()); match != nil {
			day, _ := strconv.ParseInt(match[1], 10, 32)
			month, _ := strconv.ParseInt(match[2], 10, 32)
			year, _ := strconv.ParseInt(match[3], 10, 32)
			hours, _ := strconv.ParseInt(match[4], 10, 32)
			minutes, _ := strconv.ParseInt(match[5], 10, 32)
			seconds, _ := strconv.ParseInt(match[6], 10, 32)

			tripStartTime = time.Date(int(year), time.Month(month), int(day), int(hours), int(minutes), int(seconds), 0, time.Local)
			fmt.Printf("ID: %s\n", tripId)
			fmt.Printf("start time: %s\n\n", tripStartTime)
		} else {
			fmt.Println("No Date line, it might be CAN data")
			return
		}
		buff.Reset()
	} else {
		fmt.Printf("ID: %s\n", tripId)
	}

	csvReader := csv.NewReader(data)
	debug := false

	var headerTrimRegex = regexp.MustCompile(`\s*\[.*\]$`)
	var headers = []string{}
	var headerIndex = map[string]int{}

	// parse header line
	fields, err := csvReader.Read()
	for idx, h := range fields {
		if h == "" { // all lines contains an empty field at the end
			continue
		}
		name := headerTrimRegex.ReplaceAllString(h, "")
		headers = append(headers, name)
		headerIndex[name] = idx
	}

	recordCount := 0
	// parse body lines
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
			timestamp = tripStartTime.UnixNano() + int64(rec["t"].(float64)*1000000000)
			// VALUE
			if v, ok := rec["Speed_Kmh"]; ok {
				value = v.(float64)
			}
		} else { // CAN - raw & interpolated
			// TIME
			timestamp = int64(rec["timestamps"].(float64)) * 1000000
			// VALUE
			if v, ok := rec["WHL_SpdFLVal"]; ok {
				value = v.(float64)
			}
			if debug {
				fmt.Println(rec)
				debug = false
			}
		}
		// DATA
		jsonData, err := json.Marshal(rec)
		if err != nil {
			fmt.Println("Error marshalling record", err)
			return
		}
		escaped := strings.ReplaceAll(string(jsonData), `"`, `""`)
		buff.Write([]byte(fmt.Sprintf("%s,%d,%f,\"%s\"\n", tripId, timestamp, value, escaped)))
		if debug {
			jsonData, _ = json.Marshal(rec)
			//curl -o - -H "Content-Type: text/csv" -X POST 'http://127.0.0.1:5654/db/write/HDCAR?method=insert' --data '<data>'
			fmt.Printf("%s\n", buff.String())
			debug = false
		}
		recordCount++
		// send POST request every 1000 records
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

func sendHttp(addr string, data io.Reader) {
	req, err := http.NewRequest("POST", addr, data)
	if err != nil {
		fmt.Println("Error creating request", err)
		return
	}
	req.Header.Set("Content-Type", "text/csv")
	client := http.Client{
		// uncomment this line to see the request/response
		// Transport: &loggingTransport{},
	}

	rsp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request", err)
		return
	}
	defer rsp.Body.Close()
}
