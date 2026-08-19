# BUG_REPRO：HTTP 中间件取消传播断点

## Bug 是什么
TimeoutMiddleware/TenantMiddleware 用 context.Background 而非请求 ctx，取消与租户信息被切断。

## 如何触发
```sh
go test ./internal/distribution/transport/http -run '^TestTimeoutMiddlewareCancels$' -count=1
go test ./internal/distribution/transport/http -run '^TestTenantMiddlewarePropagates$' -count=1
go test ./internal/distribution/transport/http -run '^TestWaitForReadyHonorsCancel$' -count=1
go test ./internal/distribution/transport/http -run '^TestActorFromContext$' -count=1
go test ./internal/distribution/transport/http -run '^TestRequestActor$' -count=1
go test ./internal/distribution/transport/http -run '^TestMiddlewareCtx$' -count=1
```

## 错误信息
- request context was not cancelled
- tenant not propagated
