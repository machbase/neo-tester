
## Compare machcli vs. machgo

- *2026/02/20 with api/machcli*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -rollup`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |    827/s |  12.0s     |   12.0s    |   12.0s     |
|           | rollup       |      8  |   10,000   |  6,030/s |  12.5s     |   13.2s    |   12.8s     |
|           |              |     16  |   10,000   | 10,641/s |  13.8s     |   15.0s    |   14.5s     |
|           |              |     32  |   10,000   | 15,877/s |  19.0s     |   20.1s    |   19.6s     |
|           |              |     64  |   10,000   | 18,289/s |  32.7s     |   34.9s    |   18.2s     |
|           |              |    128  |   10,000   | 17,141/s |  1m 9s     |   1m14s    |   1m13s     |
|           |              |    512  |   10,000   | 24,157/s |  3m17s     |   3m31s    |   3m28s     |


- *2026/02/20 with api/machgo with stmt-reuse*

`go run ./stockgo -c <clients> -n <per client> -h <ip> -code WISH -rollup`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stockgo      |      1  |   10,000   |  1,828/s |   5.4s     |    5.4s    |    5.4s     |
|           | rollup       |      8  |   10,000   | 11,586/s |   5.1s     |    6.9s    |    6.2s     |
|           |              |     16  |   10,000   | 22,420/s |   6.1s     |    7.1s    |    6.8s     |
|           |              |     32  |   10,000   | 28,869/s |  10.3s     |   11.0s    |   10.6s     |
|           |              |     64  |   10,000   | 29,689/s |  20.1s     |   21.5s    |   21.1s     |
|           |              |    128  |   10,000   | 29,273/s |  37.5s     |   43.6s    |   40.4s     |
|           |              |    512  |   10,000   | 29,874/s |  48.8s     |   2m51s    |   2m24s     |

- *2026/02/23 with api/machcli*

`go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -rollup`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stock        |      1  |   10,000   |    824/s |  12.1s     |   12.1s    |   12.1s     |
|           | rollup       |      8  |   10,000   |  5,946/s |  11.9s     |   13.4s    |   13.0s     |
|           |              |     16  |   10,000   | 10,512/s |  13.9s     |   15.2s    |   14.7s     |
|           |              |     32  |   10,000   | 15,898/s |  19.3s     |   20.1s    |   19.7s     |
|           |              |     64  |   10,000   | 18,314/s |  32.8s     |   34.9s    |   34.2s     |
|           |              |    128  |   10,000   | 17,043/s |  1m 7s     |   1m15s    |   1m13s     |
|           |              |    512  |   10,000   | 24,264/s |  3m14s     |   3m30s    |   3m27s     |


- *2026/02/23 with api/machgo*

`go run ./stockgo -c <clients> -n <per client> -h <ip> -code WISH -rollup`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stockgo      |      1  |   10,000   |  1,105/s |   9.0s     |    9.0s    |    9.0s     |
|           | rollup       |      8  |   10,000   |  6,777/s |  11.2s     |   11.8s    |   11.6s     |
|           |              |     16  |   10,000   | 12,847/s |  10.5s     |   12.4s    |   12.4s     |
|           |              |     32  |   10,000   | 19,333/s |  14.8s     |   16.5s    |   16.0s     |
|           |              |     64  |   10,000   | 21,031/s |  22.9s     |   29.4s    |   25.2s     |
|           |              |    128  |   10,000   | 24,891/s |  49.2s     |   51.7s    |   50.3s     |
|           |              |    512  |   10,000   | 25,132/s |  1m21s     |   3m23s    |   2m54s     |

- *2026/02/20 with api/machgo with stmt-reuse*

`go run ./stockgo -c <clients> -n <per client> -h <ip> -code WISH -rollup -reuse`
| Ver       | scenario     | clients | per client | ops/s.   | min client | max client | avg. client |
|-----------|--------------|---------|------------|----------|------------|------------|-------------|
|v8.0.73-rc3| stockgo      |      1  |   10,000   |  1,719/s |   5.8s     |    5.8s    |    5.8s     |
|           | rollup       |      8  |   10,000   | 11,443/s |   6.2s     |    6.9s    |    6.6s     |
|           | stmt-reuse   |     16  |   10,000   | 20,843/s |   5.1s     |    7.6s    |    6.4s     |
|           |              |     32  |   10,000   | 29,231/s |  10.2s     |   10.9s    |   10.6s     |
|           |              |     64  |   10,000   | 29,594/s |  21.0s     |   21.6s    |   21.2s     |
|           |              |    128  |   10,000   | 29,497/s |  35.6s     |   43.3s    |   39.1s     |
|           |              |    512  |   10,000   | 29,725/s |  44.9s     |   2m52s    |   2m24s     |

- stockbench result

```
goos: darwin
goarch: arm64
pkg: tester/stockbench
cpu: Apple M5

"github.com/machbase/neo-server/v8/api/machcli"
BenchmarkSelect-10    	      86	  14861460 ns/op	   25010 B/op	    1543 allocs/op

"github.com/machbase/neo-server/v8/api/machgo"
BenchmarkSelect-10    	     228	   4730629 ns/op	   30324 B/op	     475 allocs/op
```
