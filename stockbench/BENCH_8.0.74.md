

```
goos: linux
goarch: amd64
pkg: tester/stockbench
cpu: AMD Ryzen 9 3900X 12-Core Processor            
BenchmarkSelect_MachCli-24                    40          25689245 ns/op           41856 B/op       2248 allocs/op
BenchmarkSelect_MachCli-24                    40          27040303 ns/op           41856 B/op       2248 allocs/op
BenchmarkSelect_MachCli-24                    46          23890751 ns/op           41857 B/op       2248 allocs/op
BenchmarkSelect_MachCli-24                    49          23896260 ns/op           41857 B/op       2248 allocs/op
BenchmarkSelect_MachCli-24                    44          25323143 ns/op           41858 B/op       2248 allocs/op
BenchmarkSelect_MachGo-24                     44          24550372 ns/op           48770 B/op        853 allocs/op
BenchmarkSelect_MachGo-24                     48          23585178 ns/op           48744 B/op        853 allocs/op
BenchmarkSelect_MachGo-24                     49          24435280 ns/op           48738 B/op        853 allocs/op
BenchmarkSelect_MachGo-24                     49          24047181 ns/op           48738 B/op        853 allocs/op
BenchmarkSelect_MachGo-24                     48          23419921 ns/op           48744 B/op        853 allocs/op
BenchmarkSelectRollup_MachCli-24             949           1128820 ns/op           26560 B/op       1368 allocs/op
BenchmarkSelectRollup_MachCli-24            1166           1266434 ns/op           26560 B/op       1368 allocs/op
BenchmarkSelectRollup_MachCli-24            1179           1253205 ns/op           26560 B/op       1368 allocs/op
BenchmarkSelectRollup_MachCli-24            1030           1070670 ns/op           26560 B/op       1368 allocs/op
BenchmarkSelectRollup_MachCli-24            1078           1229429 ns/op           26560 B/op       1368 allocs/op
BenchmarkSelectRollup_MachGo-24             2005            570180 ns/op           30724 B/op        531 allocs/op
BenchmarkSelectRollup_MachGo-24             2323            489256 ns/op           30717 B/op        531 allocs/op
BenchmarkSelectRollup_MachGo-24             2030            576701 ns/op           30718 B/op        531 allocs/op
BenchmarkSelectRollup_MachGo-24             1956            570561 ns/op           30718 B/op        531 allocs/op
BenchmarkSelectRollup_MachGo-24             1968            570364 ns/op           30718 B/op        531 allocs/op
PASS
coverage: [no statements]
ok      tester/stockbench       28.930s
```

## 비교 분석 리포트 (machcli vs machgo)

아래 분석은 각 벤치마크 5회 측정값의 평균 기준으로 정리했다. (`ns/op`, `B/op`, `allocs/op`는 **낮을수록 유리**)

### 1) Select 벤치마크 (`BenchmarkSelect_*`)

| 항목 | machcli 평균 | machgo 평균 | 비교 결과 |
|---|---:|---:|---|
| `ns/op` | 25,167,940 | 24,007,586 | **machgo 4.61% 빠름** (1.048x) |
| `B/op` | 41,857 | 48,747 | **machcli 16.46% 적음** |
| `allocs/op` | 2,248 | 853 | **machgo 62.06% 적음** (machcli의 0.38x) |

해석:
- 순수 응답 시간은 `machgo`가 근소하게 우세하다.
- 하지만 바이트 사용량(`B/op`)은 `machcli`가 더 적다.
- 할당 횟수(`allocs/op`)는 `machgo`가 크게 유리하다.

### 2) Select Rollup 벤치마크 (`BenchmarkSelectRollup_*`)

| 항목 | machcli 평균 | machgo 평균 | 비교 결과 |
|---|---:|---:|---|
| `ns/op` | 1,189,712 | 555,412 | **machgo 53.32% 빠름** (2.142x) |
| `B/op` | 26,560 | 30,719 | **machcli 15.66% 적음** |
| `allocs/op` | 1,368 | 531 | **machgo 61.18% 적음** (machcli의 0.39x) |

해석:
- Rollup 시나리오에서는 `machgo`가 시간 성능에서 매우 큰 우위를 보인다.
- 메모리 바이트 사용량은 여전히 `machcli`가 더 작다.
- 할당 횟수는 `machgo`가 크게 낮아 GC 부담 측면에서도 유리하다.

### 3) 종합 결론

- **시간 성능 우선**(특히 Rollup/집계성 쿼리): `machgo` 선택이 유리.
- **전송/버퍼 바이트 사용량 최소화 우선**: `machcli`가 유리.
- 실서비스에서는 평균 성능 외에 tail latency, 동시성, 실제 payload 크기 조건을 추가 측정해 최종 선택을 권장.

요약하면, 본 결과에서는 `machgo`가 실행 시간과 할당 횟수에서 우세하고, `machcli`는 `B/op`에서 우세하다.