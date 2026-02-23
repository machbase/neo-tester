
- StatementReuse: machgo.StatementReuseAuto

```
goos: linux
goarch: amd64
pkg: tester/stockbench
cpu: AMD Ryzen 9 3900X 12-Core Processor
BenchmarkSelect_MachCli
BenchmarkSelect_MachCli-24          	     543	   2777758 ns/op	   55674 B/op	    2248 allocs/op
BenchmarkSelect_MachGo
BenchmarkSelect_MachGo-24           	     583	   2042676 ns/op	   53040 B/op	     855 allocs/op
BenchmarkSelectRollup_MachCli
BenchmarkSelectRollup_MachCli-24    	     706	   1662413 ns/op	   34864 B/op	    1368 allocs/op
BenchmarkSelectRollup_MachGo
BenchmarkSelectRollup_MachGo-24     	    1129	    963809 ns/op	   33112 B/op	     532 allocs/op
```

- StatementReuse: machgo.StatementReuseOff

```
goos: linux
goarch: amd64
pkg: tester/stockbench
cpu: AMD Ryzen 9 3900X 12-Core Processor            
BenchmarkSelect_MachCli
BenchmarkSelect_MachCli-24                   409           2806925 ns/op           55692 B/op       2248 allocs/op
BenchmarkSelect_MachGo
BenchmarkSelect_MachGo-24                    481           2463042 ns/op           59215 B/op        909 allocs/op
BenchmarkSelectRollup_MachCli
BenchmarkSelectRollup_MachCli-24             723           1624979 ns/op           34864 B/op       1368 allocs/op
BenchmarkSelectRollup_MachGo
BenchmarkSelectRollup_MachGo-24              967           1244727 ns/op           39540 B/op        586 allocs/op
```