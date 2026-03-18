-- Raw tick table: stores per-trade events per stock code with event time.
create tag table if not exists stock_tick (
    code      varchar(20) primary key,
    time      datetime basetime,
    price     double,
    volume    double,
    bid_price double,
    ask_price double
);

-- 1-second rollup target table: stores sums and count aggregated from stock_tick.
-- At query time, averages are calculated as sum/cnt to reduce full scans of raw data.
create tag table if not exists stock_rollup_1s (
    code       varchar(20) primary key,
    time       datetime basetime,
    sum_price  double,
    sum_volume double,
    sum_bid    double,
    sum_ask    double,
    cnt        integer,
    open       double,
    open_time  datetime,
    close      double,
    close_time datetime,
    high       double,
    low        double
);

-- 1-minute rollup target table: re-aggregates 1-second rollup data per minute.
-- This multi-stage structure helps reduce CPU/IO load when many events arrive between minute boundaries.
create tag table if not exists stock_rollup_1m (
    code       varchar(20) primary key,
    time       datetime basetime,
    sum_price  double,
    sum_volume double,
    sum_bid    double,
    sum_ask    double,
    cnt        integer,
    open       double,
    open_time  datetime,
    close      double,
    close_time datetime,
    high       double,
    low        double
);

-- 1-hour rollup target table: re-aggregates 1-minute rollup rows into hourly buckets.
-- This multi-stage structure helps reduce CPU/IO load for long-range queries.
create tag table if not exists stock_rollup_1h (
    code       varchar(20) primary key,
    time       datetime basetime,
    sum_price  double,
    sum_volume double,
    sum_bid    double,
    sum_ask    double,
    cnt        integer,
    open       double,
    open_time  datetime,
    close      double,
    close_time datetime,
    high       double,
    low        double
);
