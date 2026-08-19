# BUG_REPRO：复制协调器 channel 生命周期错位

## Bug 是什么
关闭后仍可入队导致 panic，Drain 计数错误，Queue.Pop 不清理去重标记，Len 偏大。

## 如何触发
```sh
go test -race ./internal/replicationctl -run '^TestEnqueueAfterCloseErrors$' -count=1
go test -race ./internal/replicationctl -run '^TestDrainCountsTasks$' -count=1
go test -race ./internal/replicationctl -run '^TestQueuePopAllowsRepush$' -count=1
go test -race ./internal/replicationctl -run '^TestQueueLen$' -count=1
go test -race ./internal/replicationctl -run '^TestCoordinatorConcurrentDrain$' -count=1
```

## 错误信息
- Enqueue after Close panicked: send on closed channel
