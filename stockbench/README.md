## 2026/08/13

goos: darwin
goarch: arm64
pkg: tester/stockbench
cpu: Apple M5
BenchmarkSelect-10          	   17361	     68732 ns/op	   11360 B/op	     109 allocs/op
BenchmarkSelectRollup-10    	   17319	     70040 ns/op	   12752 B/op	     109 allocs/op
PASS
ok  	tester/stockbench	4.066s

## 2026/08/14 (after refactoring neo-client), server engine v8.5.10

goos: darwin
goarch: arm64
pkg: tester/stockbench
cpu: Apple M5
BenchmarkSelect-10          	   37904	     32090 ns/op	    4464 B/op	      52 allocs/op
BenchmarkSelectRollup-10    	   37300	     31265 ns/op	    5104 B/op	      52 allocs/op
PASS
ok  	tester/stockbench	3.509s

## 2026/08/15 server engine { var GitHash = "1dba7e8";  var Version = "8.7" }

goos: darwin
goarch: arm64
pkg: tester/stockbench
cpu: Apple M5
BenchmarkSelect-10          	   23968	     49536 ns/op	   10512 B/op	      98 allocs/op
BenchmarkSelectRollup-10    	   19232	     54397 ns/op	   12224 B/op	      98 allocs/op
PASS
ok  	tester/stockbench	3.757s
