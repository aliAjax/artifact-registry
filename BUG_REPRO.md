# BUG_REPRO：GC 标记/清扫底层数组污染

## Bug 是什么
Sweeper.Sweep/Retained 用 s.keys[:0] 原地压缩冲掉内部 key 列表，Keys 返回共享切片，Size/Diff 计数偏大。

## 如何触发
```sh
go test ./internal/gc -run '^TestSweeperKeysSnapshot$' -count=1
go test ./internal/gc -run '^TestRetainedAfterSweep$' -count=1
go test ./internal/gc -run '^TestMarkSetSize$' -count=1
go test ./internal/gc -run '^TestSweeperDiff$' -count=1
```

## 错误信息
- Keys was mutated by Sweep
- Retained corrupted after Sweep
