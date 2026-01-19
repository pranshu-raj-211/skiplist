## Create profiles for a specific benchmark function

```bash
go test -bench=^BenchmarkInsert$ -benchmem -cpuprofile=insert_cpu.prof -memprofile=insert_mem.prof
```

## Analyze benchmarks run
```bash
go tool pprof insert_cpu.prof
```

## Benchmark with specified time to run
```bash
go test -bench=^BenchmarkInsert$ -benchtime=10s -cpuprofile=cpu.prof
```

## Prof allocated memory
```bash
go tool pprof -alloc_space mem.prof
```

