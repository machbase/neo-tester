
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


- *2026/02/20 with api/machgo*

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

- stockbench result

goos: darwin
goarch: arm64
pkg: tester/stockbench
cpu: Apple M5

"github.com/machbase/neo-server/v8/api/machcli"

BenchmarkSelect-10    	      86	  14861460 ns/op	   25010 B/op	    1543 allocs/op

"github.com/machbase/neo-server/v8/api/machgo"
BenchmarkSelect-10    	     228	   4730629 ns/op	   30324 B/op	     475 allocs/op
