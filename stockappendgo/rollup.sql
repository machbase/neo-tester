create rollup rollup_stock_1s
into (stock_rollup_1s)
as (
    select
        code,
        date_trunc('second', time) as time,
        sum(price) as sum_price,
        sum(volume) as sum_volume,
        sum(bid_price) as sum_bid,
        sum(ask_price) as sum_ask,
        count(*) as cnt,
        first(time, price) as open,
        min(time) as open_time,
        last(time, price) as close,
        max(time) as close_time,
        max(price) as high,
        min(price) as low
    from stock_tick
    group by code, time
)
interval 1 sec;

-- Roll up 1-second rollup data into 1-minute buckets.
-- Runs every 1 minute and writes into stock_rollup_1m.
create rollup rollup_stock_1m
into (stock_rollup_1m)
as (
    select
        code,
        date_trunc('minute', time) as time,
        sum(sum_price) as sum_price,
        sum(sum_volume) as sum_volume,
        sum(sum_bid) as sum_bid,
        sum(sum_ask) as sum_ask,
        sum(cnt) as cnt,
        first(open_time, open) as open,
        min(open_time) as open_time,
        last(close_time, close) as close,
        max(close_time) as close_time,
        max(high) as high,
        min(low) as low
    from stock_rollup_1s
    group by code, time
)
interval 1 min;

-- Roll up 1-minute aggregates into 1-hour buckets.
-- Runs every 1 hour and writes into stock_rollup_1h.
create rollup rollup_stock_1h
into (stock_rollup_1h)
as (
    select
        code,
        date_trunc('hour', time) as time,
        sum(sum_price) as sum_price,
        sum(sum_volume) as sum_volume,
        sum(sum_bid) as sum_bid,
        sum(sum_ask) as sum_ask,
        sum(cnt) as cnt,
        first(open_time, open) as open,
        min(open_time) as open_time,
        last(close_time, close) as close,
        max(close_time) as close_time,
        max(high) as high,
        min(low) as low
    from stock_rollup_1m
    group by code, time
)
interval 1 hour;

