
# Edge Performance Test

**Build**

```sh
go build -o perform .
```

## Table Preparation

```sql
CREATE TAG TABLE EXAMPLE (
    name    VARCHAR(50) PRIMARY KEY,
    time    DATETIME    BASETIME,
    value   DOUBLE      SUMMARIZED
) METADATA (
    lsl     DOUBLE LOWER LIMIT,
    usl     DOUBLE UPPER LIMIT 
);
```

```sql
INSERT INTO EXAMPLE metadata VALUES ('perform', 0, 100);
```

## Append Data

```sh
./perform -scenario append -database http://192.168.0.100:5654 -table example
```

## Query

```sh
./perform -scenario query -database http://192.168.0.100:5654 -table example -time 1749527183971487000
```

## LSL

Append 과정에 1/2 지점에서 LSL 값을 1건을 의도적으로 입력

```sh
./perform -scenario lsl -database http://192.168.0.100:5654 -table example
```

## USL

Append 과정에 1/2 지점에서 USL 값을 1건을 의도적으로 입력

```sh
./perform -scenario usl -database http://192.168.0.100:5654 -table example
```