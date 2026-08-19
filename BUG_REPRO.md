# BUG_REPRO：quota 状态机错位

## Bug 是什么
Ledger.Put 把 reserved 当成无条件创建，转换后写回旧状态，Meter.Add 越限判定边界错误，CountByState 偏大。

## 如何触发
```sh
go test ./internal/quota -run '^TestMeterAddOverLimit$' -count=1
go test ./internal/quota -run '^TestLedgerRejectsBadMove$' -count=1
go test ./internal/quota -run '^TestLedgerPersistsState$' -count=1
go test ./internal/quota -run '^TestLedgerCountByState$' -count=1
```

## 错误信息
- committed -> reserved transition should be rejected
