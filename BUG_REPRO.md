# BUG_REPRO：应用层错误链断裂

## Bug 是什么
应用层 GetManifest/CreateRepository/SubmitScan 用 %v 包装 domain sentinel，errors.Is 无法识别 ErrManifestNotFound/ErrManifestBlocked/ErrNotImplemented。

## 如何触发
```sh
go test ./internal/application -run '^TestGetManifestWrapsNotFound$' -count=1
go test ./internal/application -run '^TestGetManifestWrapsBlocked$' -count=1
go test ./internal/application -run '^TestCreateRepositoryWrapsInvalidName$' -count=1
go test ./internal/application -run '^TestSubmitScanWrapsNotImplemented$' -count=1
```

## 错误信息
- expected ErrManifestNotFound, got ...
