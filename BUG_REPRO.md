# BUG_REPRO：零值构造 nil map panic

## Bug 是什么
零值 Manifest/ReferenceGraph 未惰性初始化 Annotations/Edges/Reverse，写入 nil map 直接 panic。

## 如何触发
```sh
go test ./internal/domain -run '^TestZeroValueManifestPutOne$' -count=1
go test ./internal/domain -run '^TestZeroValueManifestAddTags$' -count=1
go test ./internal/domain -run '^TestZeroValueReferenceGraphAddManifest$' -count=1
go test ./internal/domain -run '^TestZeroValueReferenceGraphRemove$' -count=1
```

## 错误信息
- panic: assignment to entry in nil map
