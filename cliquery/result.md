conn.Query()

$ go run ./cliquery -h 192.168.0.90 -p 33000 -c 100 -n 1000
Neo Engine Version: 8.0-test Build: 7b19097
All clients (100) query(1000) completed in 5.976154322s  16733 ops/sec
  Sessions: min 4.79513744s, max 5.777529803s, avg 5.537888508s

$ go run ./cliquery -h 192.168.0.90 -p 33000 -c 200 -n 1000
Neo Engine Version: 8.0-test Build: 7b19097
All clients (200) query(1000) completed in 10.54215136s  18971 ops/sec
  Sessions: min 9.096118261s, max 10.441546409s, avg 9.971498544s

$ go run ./cliquery -h 192.168.0.90 -p 33000 -c 500 -n 1000
Neo Engine Version: 8.0-test Build: 7b19097
All clients (500) query(1000) completed in 20.464279191s  24432 ops/sec
  Sessions: min 17.042806464s, max 20.215271412s, avg 19.424377887s


conn.Prepare()

$ go run ./cliquery -h 192.168.0.90 -p 33000 -c 100 -n 1000
Neo Engine Version: 8.0-test Build: 7b19097
All clients (100) query(1000) completed in 3.607562929s  27719 ops/sec
  Sessions: min 3.271580296s, max 3.596857723s, avg 3.495757465s

$ go run ./cliquery -h 192.168.0.90 -p 33000 -c 200 -n 1000
Neo Engine Version: 8.0-test Build: 7b19097
All clients (200) query(1000) completed in 7.010409904s  28529 ops/sec
  Sessions: min 4.424554662s, max 6.99081897s, avg 6.035356331s

$ go run ./cliquery -h 192.168.0.90 -p 33000 -c 500 -n 1000
Neo Engine Version: 8.0-test Build: 7b19097
All clients (500) query(1000) completed in 17.417238803s  28707 ops/sec
  Sessions: min 4.689627255s, max 17.29675149s, avg 14.512095561s