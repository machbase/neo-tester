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
	atStartTime := 1
	offset := 0
	timeForamt := "ns" // s(unix), ms(unix), us(unix), ns(unix)

	flag.StringVar(&serverAddr, "server", serverAddr, "Server address")
	flag.StringVar(&inputFile, "in", inputFile, "Input file")
	flag.IntVar(&offset, "offset", offset, "Input file line offset")
	flag.IntVar(&atStartTime, "start-time", atStartTime, "Trip start time line number, if not CAN data")
	flag.BoolVar(&isCanData, "can", isCanData, "CAN data")
	flag.StringVar(&timeForamt, "ts", timeForamt, "timestamp format, if CAN data, format: n, ms, us, ns")
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
	var tripStartTime time.Time
	reader := bufio.NewReader(data)
	for i := 0; i < offset; i++ {
		str, err := reader.ReadString('\n')
		if err != nil {
			return // EOF
		} else {
			if !isCanData && i == atStartTime-1 {
				// read the first line which contains the trip start time
				// ex) Date:06.04.2023 Time:06:57:39

				// parse start time
				var startTimeRegex = regexp.MustCompile(`Date:(\d{2})\.(\d{2})\.(\d{4}) Time:(\d{2}):(\d{2}):(\d{2})`)
				if match := startTimeRegex.FindStringSubmatch(str); match != nil {
					day, _ := strconv.ParseInt(match[1], 10, 32)
					month, _ := strconv.ParseInt(match[2], 10, 32)
					year, _ := strconv.ParseInt(match[3], 10, 32)
					hours, _ := strconv.ParseInt(match[4], 10, 32)
					minutes, _ := strconv.ParseInt(match[5], 10, 32)
					seconds, _ := strconv.ParseInt(match[6], 10, 32)

					tripStartTime = time.Date(int(year), time.Month(month), int(day), int(hours), int(minutes), int(seconds), 0, time.Local)
					fmt.Printf("start time: %s\n\n", tripStartTime)
				} else {
					fmt.Println("No Date line, it might be CAN data")
					return
				}
			}

			offset--
		}
	}

	// parse header line
	var headerTrimRegex = regexp.MustCompile(`\s*\[.*\]$`)
	var headers = []string{}
	var headerIndex = map[string]int{}

	csvReader := csv.NewReader(reader)
	fields, err := csvReader.Read()
	if err != nil {
		fmt.Println("Error reading header", err)
		return
	}

	for idx, h := range fields {
		if h == "" { // all lines contains an empty field at the end
			continue
		}
		name := headerTrimRegex.ReplaceAllString(h, "")
		headers = append(headers, name)
		headerIndex[name] = idx
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
			timestamp = tripStartTime.UnixNano() + int64(rec["t"].(float64)*1000000000)
			// VALUE
			if v, ok := rec["Speed_Kmh"]; ok {
				value = v.(float64)
			}
		} else { // CAN - raw & interpolated
			// TIME
			timestamp = int64(rec["timestamps"].(float64))
			switch timeForamt {
			case "s":
				timestamp = timestamp * 1_000_000_000
			case "ms":
				timestamp = timestamp * 1_000_000
			case "us":
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
