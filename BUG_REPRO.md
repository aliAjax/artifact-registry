# BUG_REPRO：仓储读路径缺锁 + 内部指针逃逸

## Bug 是什么
Catalog Get 类方法不加读锁并返回内部指针，并发读写产生竞态。

## 如何触发
```sh
go test -race ./internal/memory -run '^TestGetDoesNotAlias$' -count=1
go test -race ./internal/memory -run '^TestGetBlobDoesNotAlias$' -count=1
go test -race ./internal/memory -run '^TestGetTagDoesNotAlias$' -count=1
go test -race ./internal/memory -run '^TestGetUploadDoesNotAlias$' -count=1
go test -race ./internal/memory -run '^TestCatalogConcurrentAccess$' -count=1
```

## 错误信息
- WARNING: DATA RACE
