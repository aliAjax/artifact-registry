# BUG_REPRO：观测指标快照底层共享

## Bug 是什么
Histogram.Values/Quantile 共享底层数组，Span.AttributesCopy 返回共享 map。

## 如何触发
```sh
go test ./internal/platform/observability -run '^TestHistogramValuesIsolated$' -count=1
go test ./internal/platform/observability -run '^TestHistogramQuantileCopy$' -count=1
go test ./internal/platform/observability -run '^TestSpanAttributesCopy$' -count=1
go test ./internal/platform/observability -run '^TestSpanAttributeKeys$' -count=1
```

## 错误信息
- Values was mutated by Quantile
- Quantile mutated internal order
- AttributesCopy shared the attributes map
