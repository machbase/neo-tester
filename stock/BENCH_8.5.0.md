
## v8.5.0 + (fix inline view - https://github.com/machbase/dbms-nfx/issues/3624)

- APPEND:
    `go run ./stockappend -h 192.168.0.90 -p 35656 -tps 10000 -create`

- v8.5.1-rc1  *2026/04/29*

`go run ./stock -c 1 -n 10000 -h 192.168.0.90 -p 35656 -code WISH -union 1s`

| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.5.0     | stock        |      1  |   10,000   |  1,196/s |    8.3s    |    8.3s    |     8.3s    |
|           | union        |      8  |   10,000   |  8,178/s |    6.5s    |    9.7s    |     9.7s    |
|           |   (1m, 1s)   |     16  |   10,000   | 14,331/s |    7.0s    |   11.1s    |     9.9s    |
|           |              |     32  |   10,000   | 28,290/s |    7.2s    |   11.2s    |     9.4s    |
|           |              |     64  |   10,000   | 48,121/s |   12.1s    |   12.9s    |    12.6s    |
|           |              |    128  |   10,000   | 49,518/s |   24.6s    |   25.6s    |    25.2s    |
|           |              |    512  |   10,000   | 49,494/s |   44.1s    |   1m42s    |    1m26s    |

## v8.7.1 bench

### Server CPU Affinity

- neo-start.sh

```sh
#!/bin/bash
exec taskset -c 0-35 /home/eirny/machbase-neo-v8.7.1-linux-amd64/machbase-neo serve \
	--daemon \
	--pid /home/eirny/neo.pid \
	--log-filename /home/eirny/log/neo.log \
	--shell-port 35652 --mqtt-port 35653 --http-port 35654 --mach-port 35656 \
	--host 0.0.0.0
```

- neo-stop.sh

```sh
#!/bin/bash
PID=`cat /home/eirny/neo.pid`
kill $PID
while kill -0 $PID 2>/dev/null; do
  sleep 1
done
```

### Background Appender

```sh
go run ./stockappend -h 192.168.0.90 -p 35656 -tps 10000 -create
```

### Query Bench Result

- v8.5.12

`go run ./stock -c 1 -n 10000 -h 192.168.0.90 -p 35656 -code WISH -union 1s`

| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.5.12    | stock        |      1  |   10,000   |  1,780/s |    5.6s    |    5.6s    |     5.6s    |
|           | union        |      8  |   10,000   | 13,204/s |    5.9s    |    6.0s    |     6.0s    |
|           |   (1m, 1s)   |     16  |   10,000   | 26,140/s |    6.0s    |    6.1s    |     6.0s    |
|           |              |     32  |   10,000   | 47,885/s |    6.5s    |    6.6s    |     6.5s    |
|           |              |     64  |   10,000   | 73,903/s |    8.4s    |    8.6s    |     8.5s    |
|           |              |    128  |   10,000   | 85,878/s |   14.3s    |   14.8s    |    14.6s    |
|           |              |    512  |   10,000   | 81,107/s |   1m 0s    |   1m 2s    |    1m 2s    |

- v8.7.1

`go run ./stock -c 1 -n 10000 -h 192.168.0.90 -p 35656 -code WISH -union 1s`

| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.7.1     | stock        |      1  |   10,000   |  1,036/s |    9.6s    |    9.6s    |     9.6s    |
|           | union        |      8  |   10,000   |  6,351/s |   12.4s    |   12.5s    |    12.5s    |
|           |   (1m, 1s)   |     16  |   10,000   | 12,203/s |   12.8s    |   13.1s    |    13.0s    |
|           |              |     32  |   10,000   | 20,833/s |   15.1s    |   15.3s    |    15.2s    |
|           |              |     64  |   10,000   | 28,194/s |   22.5s    |   22.6s    |    22.6s    |
|           |              |    128  |   10,000   | 28,674/s |   44.3s    |   44.5s    |    44.4s    |
|           |              |    512  |   10,000   | 28,654/s |   2m41s    |   2m58s    |    2m53s    |
