
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

### Query Bench Result "Localhost"

`taskset -c 36-63 go run ./stock -c 1 -n 10000 -h 127.0.0.1 -p 35656 -code WISH -union 1s`

- v8.5.12 from localhost

| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.5.12    | stock        |      1  |   10,000   |  1,794/s |    5.5s    |    5.5s    |     5.5s    |
|           | union        |      8  |   10,000   | 12,352/s |    6.4s    |    6.4s    |     6.4s    |
|           |   (1m, 1s)   |     16  |   10,000   | 20,937/s |    7.5s    |    7.6s    |     7.5s    |
|           |              |     32  |   10,000   | 38,359/s |    8.1s    |    8.3s    |     8.2s    |
|           |              |     64  |   10,000   | 53,888/s |   11.5s    |   11.8s    |    11.7s    |
|           |              |    128  |   10,000   | 59,412/s |   19.7s    |   21.4s    |    20.8s    |
|           |              |    512  |   10,000   | 57,021/s |   1m25s    |   1m29s    |    1m28s    |

- v8.7.1 from localhost

| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.7.1     | stock        |      1  |   10,000   |  1,611/s |    6.2s    |    6.2s    |     6.2s    |
|           | union        |      8  |   10,000   |  9,687/s |    8.1s    |    8.2s    |     8.2s    |
|           |   (1m, 1s)   |     16  |   10,000   | 18,014/s |    9.7s    |    8.8s    |     8.8s    |
|           |              |     32  |   10,000   | 31,854/s |    9.8s    |   10.0s    |     9.9s    |
|           |              |     64  |   10,000   | 42,979/s |   14.5s    |   14.8s    |    14.7s    |
|           |              |    128  |   10,000   | 49,122/s |   25.2s    |   25.9s    |    25.7s    |
|           |              |    512  |   10,000   | 51,950/s |   1m35s    |   1m38s    |    1m37s    |


### Query Bench Result "Remote"

- v8.5.12 from remote

| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.5.12    | stock        |      1  |   10,000   |  1,271/s |    7.8s    |    7.8s    |     7.8s    |
|           | union        |      8  |   10,000   |  9,032/s |    8.6s    |    8.8s    |     8.7s    |
|           |   (1m, 1s)   |     16  |   10,000   | 16,734/s |    9.4s    |    9.5s    |     9.4s    |
|           |              |     32  |   10,000   | 31,052/s |   10.1s    |   10.2s    |    10.2s    |
|           |              |     64  |   10,000   | 33,107/s |   19.1s    |   19.2s    |    19.1s    |
|           |              |    128  |   10,000   | 33,032/s |   37.1s    |   38.6s    |    38.1s    |
|           |              |    512  |   10,000   | 33,223/s |   2m26s    |   2m33s    |    2m32s    |

- v8.7.1 from remote

| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.7.1     | stock        |      1  |   10,000   |  1,030/s |    9.7s    |    9.7s    |     9.7s    |
|           | union        |      8  |   10,000   |  6,716/s |   11.7s    |   11.8s    |    11.8s    |
|           |   (1m, 1s)   |     16  |   10,000   | 12,669/s |   12.3s    |   12.6s    |    12.4s    |
|           |              |     32  |   10,000   | 21,547/s |   14.5s    |   14.8s    |    14.6s    |
|           |              |     64  |   10,000   | 29,792/s |   21.3s    |   21.4s    |    21.3s    |
|           |              |    128  |   10,000   | 30,145/s |   42.1s    |   42.3s    |    42.2s    |
|           |              |    512  |   10,000   | 29,092/s |   2m55s    |   2m55s    |    2m51s    |
