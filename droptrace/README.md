# DROP CASCADE trace

Trace the client result, table existence, row count, and directly connected
rollup names before and after `DROP TABLE ... CASCADE`.
Metadata checks include the owner, so SYS and DEMO tables with the same name
are not confused. Snapshot errors stop the trace rather than producing a verdict.

## Cross-user reproduction

Run these commands from `neo-tester`. Use only disposable test tables.
The stockappend schema contains four tables and three custom rollups.

```sh
export NEOSHELL_HOST=127.0.0.1:5654
export NEOSHELL_USER=sys
export NEOSHELL_PASSWORD=manager
../neo-server/tmp/machbase-neo shell sql 'create user demo identified by pass'

go run ./stockappend -u demo -P pass -tps 100000 -create
```

In another terminal, drop the DEMO table as SYS:

```sh
go run ./droptrace -u sys -P manager -table demo.stock_tick -via shell
go run ./droptrace -u sys -P manager -table demo.stock_tick -via query
go run ./droptrace -u sys -P manager -table demo.stock_tick -via exec
```

For an exact tagged shell build, pass `-neo /path/to/machbase-neo`.
The shell uses the HTTP port (`-http-port`, default 5654);
direct driver operations use `-p` (default 5656).

For an interactive check, log into `machbase-neo shell` as SYS and execute:

```sql
drop table demo.stock_tick cascade;
select count(*) from demo.stock_tick;
```

A failed DROP can remove rollup definitions even though the source table
remains. Each subsequent attempt therefore starts from a different rollup state.
Stop stockappend, then drop all four DEMO tables with CASCADE before using
`stockappend -create` again: `stock_tick`, `stock_rollup_1s`,
`stock_rollup_1m`, and `stock_rollup_1h`.

## Observed result (2026-10-28)

Using a shell built from neo-server tag `v8.5.12` against the local server
with engine signature `8.5.12.official-DARWIN-ARM_M1-64-release-standard`:

- SYS executed `create user demo identified by pass`. DEMO created all four
  tables and all three custom rollups (owner DEMO in `M$SYS_TABLES` and
  `V$ROLLUP`); stockappend authenticated as DEMO at about 200,000 rows/second.
- The first `drop table demo.stock_tick cascade;` was executed by SYS in the
  interactive shell. It printed `MACHCLI-ERR-2031, Resource busy (STOCK_TICK).`
  The source table remained (row count kept increasing), but its directly
  connected `ROLLUP_STOCK_1S` disappeared; `ROLLUP_STOCK_1M/1H` remained.
- Repeated attempts through the interactive shell, `shell sql` (exit code 1),
  and direct query/exec paths returned the same error.
- `drop table stock_tick cascade` by SYS without an owner prefix returned
  `MACHCLI-ERR-2025, Table STOCK_TICK does not exist.`
- After stopping stockappend, SYS dropped DEMO.STOCK_TICK on the first
  attempt (`table dropped.`, exit code 0, about 3.1 s waiting for the rollup).

The missing-error symptom was not reproduced in this cross-user scenario.
