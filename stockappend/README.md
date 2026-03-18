
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

