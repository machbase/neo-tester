
## v8.5.0 + (fix inline view - https://github.com/machbase/dbms-nfx/issues/3624)

- APPEND:
    `go run ./stockappend -h 192.168.0.90 -p 35656 -tps 10000 -create`

- *2025/04/29* release candiate v8.5.1

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

