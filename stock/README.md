# stock data test

## Create table and append data

- Use `stockappend` for creating tables and generating data.

```sh
go run ./stockappend -h [host] -p [port] -u [username] -P [password] -tps [tps_number]
```

- `-h`, `-p`, `-u`, `-P` machbase account
- `-tps`: append TPS (e.g., 5 = 20ms interval) (default 1000)


## Query performance

- Use `stock` for measuring query performance

```sh
go run ./stock -h [host] -p [port] -u [username] -P [password] <...options>
```

- `-h`, `-p`, `-u`, `-P` machbase account
- `-c int` number of clients (default 50)
- `-n int` number of queries per client (default 1000)
- `-code string` stock code (tag) to insert/query (default "AAPL") --> Refer [Codes List](../stockappend/stock_codes.txt)
- `-rollup` perform rollup query instead of tick query

- `-f int` number of rows to fetch per query (default 100) --> translated in `limit <int>`
- `-prep` re-use prepared statement --> one statement per a client.
- `-prof` enable Go cpu profiling
- `-T` enable OS thread lock

ex)

```
go run ./stock -c <clients> -n <per client> -h <ip> -code WISH
```

```
go run ./stock -c <clients> -n <per client> -h <ip> -code WISH -rollup
```

### Jemalloc

Use build tag `-tags debug`.

ex)

```
go run -tags debug ./stock -c <clients> -n <per client> -h <ip> -code WISH
```

```
go build -tags debug -o /tmp/stock ./stock
```
