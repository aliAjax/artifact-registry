# BUG_REPRO：扫描引擎 defer 吞错

## Bug 是什么
Engine.Scan/ScanAll/Report.Merge/Combine/ValidateFindings 用命名返回 + defer 吞掉错误。

## 如何触发
```sh
go test ./internal/scanner -run '^TestScanReturnsError$' -count=1
go test ./internal/scanner -run '^TestScanAllReturnsError$' -count=1
go test ./internal/scanner -run '^TestMergeRejectsDifferentArtifacts$' -count=1
go test ./internal/scanner -run '^TestCombineRejectsDifferentArtifacts$' -count=1
go test ./internal/scanner -run '^TestValidateFindingsReturnsError$' -count=1
```

## 错误信息
- expected scan error to be returned
