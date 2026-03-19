
## Compare machcli vs. machgo

- *2026/02/20 with api/machgo with stmt-reuse*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -rollup`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |  1,828/s |   5.4s     |    5.4s    |    5.4s     |
|           | rollup       |      8  |   10,000   | 11,586/s |   5.1s     |    6.9s    |    6.2s     |
|           |              |     16  |   10,000   | 22,420/s |   6.1s     |    7.1s    |    6.8s     |
|           |              |     32  |   10,000   | 28,869/s |  10.3s     |   11.0s    |   10.6s     |
|           |              |     64  |   10,000   | 29,689/s |  20.1s     |   21.5s    |   21.1s     |
|           |              |    128  |   10,000   | 29,273/s |  37.5s     |   43.6s    |   40.4s     |
|           |              |    512  |   10,000   | 29,874/s |  48.8s     |   2m51s    |   2m24s     |


- *2026/02/23 with api/machgo*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -rollup`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |  1,105/s |   9.0s     |    9.0s    |    9.0s     |
|           | rollup       |      8  |   10,000   |  6,777/s |  11.2s     |   11.8s    |   11.6s     |
|           |              |     16  |   10,000   | 12,847/s |  10.5s     |   12.4s    |   12.4s     |
|           |              |     32  |   10,000   | 19,333/s |  14.8s     |   16.5s    |   16.0s     |
|           |              |     64  |   10,000   | 21,031/s |  22.9s     |   29.4s    |   25.2s     |
|           |              |    128  |   10,000   | 24,891/s |  49.2s     |   51.7s    |   50.3s     |
|           |              |    512  |   10,000   | 25,132/s |  1m21s     |   3m23s    |   2m54s     |

- *2026/02/20 with api/machgo with stmt-reuse*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -rollup -reuse`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |  1,719/s |   5.8s     |    5.8s    |    5.8s     |
|           | rollup       |      8  |   10,000   | 11,443/s |   6.2s     |    6.9s    |    6.6s     |
|           | stmt-reuse   |     16  |   10,000   | 20,843/s |   5.1s     |    7.6s    |    6.4s     |
|           |              |     32  |   10,000   | 29,231/s |  10.2s     |   10.9s    |   10.6s     |
|           |              |     64  |   10,000   | 29,594/s |  21.0s     |   21.6s    |   21.2s     |
|           |              |    128  |   10,000   | 29,497/s |  35.6s     |   43.3s    |   39.1s     |
|           |              |    512  |   10,000   | 29,725/s |  44.9s     |   2m52s    |   2m24s     |


- *2026/02/24 with api/machgo  union query*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -union`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |    573/s |  17.4s     |   17.4s    |   17.4s     |
|           | union        |      8  |   10,000   |  4,727/s |  16.3s     |   16.9s    |   16.6s     |
|           |              |     16  |   10,000   |  8,381/s |  18.5s     |   19.0s    |   18.8s     |
|           |              |     32  |   10,000   | 17,860/s |  17.5s     |   17.9s    |   17.7s     |
|           |              |     64  |   10,000   | 21,519/s |  22.0s     |   29.6s    |   27.4s     |
|           |              |    128  |   10,000   | 20,085/s |  54.2s     |   1m03s    |   1m01s     |
|           |              |    512  |   10,000   | 19,835/s |  4m01s     |   4m17s    |   4m15s     |

- *2026/02/25 with api/machgo  union query, after fix UNION ALL double and AVG(), union(1m, tick)*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -union`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |    590/s |  16.9s     |   16.9s    |   16.9s     |
|           | union        |      8  |   10,000   |  4,791/s |  16.1s     |   16.6s    |   16.3s     |
|           |   (1m, tick) |     16  |   10,000   |  9,819/s |  15.8s     |   16.2s    |   15.9s     |
|           |              |     32  |   10,000   | 19,105/s |  16.2s     |   16.7s    |   16.4s     |
|           |              |     64  |   10,000   | 20,489/s |  25.8s     |   31.2s    |   29.7s     |
|           |              |    128  |   10,000   | 22,433/s |  47.8s     |   56.9s    |   55.4s     |
|           |              |    512  |   10,000   | 22,752/s |  3m03s     |   3m44s    |   3m40s     |

- *2026/02/25 with api/machgo  union query, after fix UNION ALL double and AVG(), union(1m, 1s)*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -union`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |  1,168/s |   8.5s     |    8.5s    |    8.5s     |
|           | union        |      8  |   10,000   |  6,914/s |   8.9s     |   11.5s    |    9.5s     |
|           |   (1m, 1s)   |     16  |   10,000   | 13,709/s |   9.7s     |   11.6s    |   11.1s     |
|           |              |     32  |   10,000   | 24,219/s |   8.3s     |   13.2s    |   10.6s     |
|           |              |     64  |   10,000   | 32,363/s |  18.4s     |   19.7s    |   19.0s     |
|           |              |    128  |   10,000   | 32,791/s |  34.2s     |   39.0s    |   37.0s     |
|           |              |    512  |   10,000   | 33,215/s |  45.0s     |   2m33s    |   2m12s     |
