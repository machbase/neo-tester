select name, (conflict_count * 10000 / (try_count + 1) / 100) as rate, read_try_count, write_try_count, try_count, conflict_count, wait_msec, wait_avg_msec, held_msec, held_avg_msec from v$mutex order by conflict_count desc limit 100;

select name, (conflict_count * 10000 / (try_count + 1) / 100) as rate, try_count, conflict_count, wait_msec, held_msec from v$mutex order by conflict_count desc limit 10;
