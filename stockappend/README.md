
## How to run

```sh
go run ./stockappend -tps 500000 -create
```


- `-create` : create tables and rollups

## Outputs

```
2026-03-18 21:59:25 TPS: 499,607/s Read: 0B (0B/s), Write: 190.7MB (38.1MB/s) <- append performance per 5s
  ROLLUP_STOCK_1S max elapsed: 855ms, max gap: 175,951                        <- rollup gap per 1s
  ROLLUP_STOCK_1S max elapsed: 927ms, max gap: 175,387
  ROLLUP_STOCK_1S max elapsed: 847ms, max gap: 175,661
  ROLLUP_STOCK_1S max elapsed: 891ms, max gap: 163,187
  ROLLUP_STOCK_1S max elapsed: 796ms, max gap: 175,526
2026-03-18 21:59:30 TPS: 500,076/s Read: 0B (0B/s), Write: 190.8MB (38.2MB/s)
  ROLLUP_STOCK_1S max elapsed: 820ms, max gap: 175,668
  ROLLUP_STOCK_1S max elapsed: 930ms, max gap: 175,471
  ROLLUP_STOCK_1S max elapsed: 861ms, max gap: 163,388
  ROLLUP_STOCK_1S max elapsed: 812ms, max gap: 175,578
  ROLLUP_STOCK_1S max elapsed: 866ms, max gap: 163,469
```

### v8.5.0 + (fix inline view - https://github.com/machbase/dbms-nfx/issues/3624)

```sh
go run ./stockappend -h 192.168.0.90 -p 35656 -tps 500000 -create
```

2026-04-29 12:36:25 TPS: 500,000/s Read: 0B (0B/s), Write: 190.8MB (38.2MB/s)
2026-04-29 12:36:26 Rollup ROLLUP_STOCK_1S elapsed: 275.12ms, gap: 108,553
2026-04-29 12:36:27 Rollup ROLLUP_STOCK_1S elapsed: 258.71ms, gap: 107,806
2026-04-29 12:36:28 Rollup ROLLUP_STOCK_1S elapsed: 375.58ms, gap: 107,827
2026-04-29 12:36:29 Rollup ROLLUP_STOCK_1S elapsed: 265.36ms, gap: 156,341
2026-04-29 12:36:30 Rollup ROLLUP_STOCK_1S elapsed: 270.96ms, gap: 282,107
2026-04-29 12:36:30 TPS: 500,086/s Read: 0B (0B/s), Write: 190.9MB (38.2MB/s)
2026-04-29 12:36:31 Rollup ROLLUP_STOCK_1S elapsed: 631.93ms, gap: 408,103
2026-04-29 12:36:32 Rollup ROLLUP_STOCK_1S elapsed: 293.54ms, gap: 534,118
2026-04-29 12:36:33 Rollup ROLLUP_STOCK_1S elapsed: 343.52ms, gap: 660,343
2026-04-29 12:36:34 Rollup ROLLUP_STOCK_1S elapsed: 238.94ms, gap: 786,449
2026-04-29 12:36:35 Rollup ROLLUP_STOCK_1S elapsed: 218.11ms, gap: 912,752