import json, pathlib
OUT = pathlib.Path('/Users/hutu/Desktop/go_projects/work_projects/001-artifact-registry/2026-08-20/artifact-registry/.work')
GO = "golang:1.23; go 1.23.0; GOTOOLCHAIN=local"
R = {}
def rec(record, sample_id, bug_id, task_type, category, uq, vc, grc, sc, contract):
    R[record] = {"sample_id": sample_id, "session_id": "", "bug_id": bug_id, "task_type": task_type,
        "bug_category": category, "repo_url": "", "go_version": GO, "repro_determinism": "deterministic",
        "user_query": uq, "trajectory": "", "verify_cmds": vc, "gold_root_cause": grc, "gold_patch": "",
        "success_criteria": sc, "verify_result": "", "harness": "", "generator_model": "", "worker": "",
        "creator": "", "qc_result": "", "qc_note": "", "sync_feishu": "", "coverage_contract": contract}

rec("001", 201, "ar-error-wrap-manifest-001", "bugfix", "error异常错误",
"""拉 manifest 和建仓库时报的错用 errors.Is 判不到 domain 里的 sentinel，阻断的 manifest 也不报 ErrManifestBlocked。代码就在当前目录，帮我修好。""",
"""go test ./internal/artifact/application -run '^TestGetManifestWrapsNotFound$' -count=1
go test ./internal/artifact/application -run '^TestGetManifestWrapsBlocked$' -count=1
go test ./internal/artifact/application -run '^TestCreateRepositoryWrapsInvalidName$' -count=1
go test ./internal/artifact/application -run '^TestSubmitScanWrapsNotImplemented$' -count=1""",
"""文件: internal/artifact/application/registry.go, internal/artifact/application/lifecycle.go 符号: RegistryService.GetManifest, LifecycleService.SubmitScan 机制: 应用层用 %v 而非 %w 包装 domain sentinel，errors.Is 无法识别 ErrManifestNotFound/ErrManifestBlocked/ErrNotImplemented""",
"""1. GetManifest 访问不存在的仓库要能被 errors.Is 认成 ErrManifestNotFound，阻断的 manifest 要能被认成 ErrManifestBlocked；2. 修完后这四条连续 20 遍全绿；3. 撤回任何一处 %w 包装，对应测试立刻又红；4. 全量测试无回归，只验证公开行为。""",
{"version":1,"claims":[
 {"id":"get_manifest_wraps_not_found","description":"GetManifest 包装 ErrManifestNotFound","test":"TestGetManifestWrapsNotFound","command":"go test ./internal/artifact/application -run '^TestGetManifestWrapsNotFound$' -count=1","paths":["internal/artifact/application/registry.go"]},
 {"id":"get_manifest_wraps_blocked","description":"GetManifest 包装 ErrManifestBlocked","test":"TestGetManifestWrapsBlocked","command":"go test ./internal/artifact/application -run '^TestGetManifestWrapsBlocked$' -count=1","paths":["internal/artifact/application/registry.go"]},
 {"id":"create_repository_wraps_invalid","description":"CreateRepository 包装 ErrInvalidRepository","test":"TestCreateRepositoryWrapsInvalidName","command":"go test ./internal/artifact/application -run '^TestCreateRepositoryWrapsInvalidName$' -count=1","paths":["internal/artifact/application/registry.go"]},
 {"id":"submit_scan_wraps_not_impl","description":"SubmitScan 包装 ErrNotImplemented","test":"TestSubmitScanWrapsNotImplemented","command":"go test ./internal/artifact/application -run '^TestSubmitScanWrapsNotImplemented$' -count=1","paths":["internal/artifact/application/lifecycle.go"]}]})

rec("002", 202, "ar-nil-zero-manifest-002", "bugfix", "nil相关问题",
"""零值的 Manifest 一调 SetAnnotation 就崩，ReferenceGraph 零值跑 AddManifest 也崩。贴一段：\n--- FAIL: TestZeroValueManifestSetAnnotation\n    nil_test.go:9: panicked: assignment to entry in nil map\n麻烦帮我修好。""",
"""go test ./internal/artifact/domain -run '^TestZeroValueManifestSetAnnotation$' -count=1
go test ./internal/artifact/domain -run '^TestZeroValueManifestAddAnnotations$' -count=1
go test ./internal/artifact/domain -run '^TestZeroValueReferenceGraphAddManifest$' -count=1
go test ./internal/artifact/domain -run '^TestZeroValueReferenceGraphReachable$' -count=1
go test ./internal/artifact/domain -run '^TestZeroValueReferenceGraphRemove$' -count=1""",
"""文件: internal/artifact/domain/manifest.go, internal/artifact/domain/validation.go 符号: Manifest.SetAnnotation, ReferenceGraph.AddManifest 机制: 零值构造不惰性初始化 Annotations/Edges/Reverse map，写入 nil map 直接 panic""",
"""1. 零值 Manifest 的 SetAnnotation/AddAnnotations 不 panic，零值 ReferenceGraph 的 AddManifest/Reachable/Remove 也不 panic；2. 修好后这五条连续 20 遍全绿；3. 撤掉任一惰性初始化，对应测试立刻又 panic 变红；4. 全量测试无回归，只验证公开行为。""",
{"version":1,"claims":[
 {"id":"zero_manifest_set_annotation","description":"零值 Manifest.SetAnnotation 不 panic","test":"TestZeroValueManifestSetAnnotation","command":"go test ./internal/artifact/domain -run '^TestZeroValueManifestSetAnnotation$' -count=1","paths":["internal/artifact/domain/manifest.go"]},
 {"id":"zero_manifest_add_annotations","description":"零值 Manifest.AddAnnotations 不 panic","test":"TestZeroValueManifestAddAnnotations","command":"go test ./internal/artifact/domain -run '^TestZeroValueManifestAddAnnotations$' -count=1","paths":["internal/artifact/domain/manifest.go"]},
 {"id":"zero_graph_add_manifest","description":"零值 ReferenceGraph.AddManifest 不 panic","test":"TestZeroValueReferenceGraphAddManifest","command":"go test ./internal/artifact/domain -run '^TestZeroValueReferenceGraphAddManifest$' -count=1","paths":["internal/artifact/domain/validation.go"]},
 {"id":"zero_graph_reachable","description":"零值 ReferenceGraph.Reachable 不 panic","test":"TestZeroValueReferenceGraphReachable","command":"go test ./internal/artifact/domain -run '^TestZeroValueReferenceGraphReachable$' -count=1","paths":["internal/artifact/domain/validation.go"]},
 {"id":"zero_graph_remove","description":"零值 ReferenceGraph.Remove 不 panic","test":"TestZeroValueReferenceGraphRemove","command":"go test ./internal/artifact/domain -run '^TestZeroValueReferenceGraphRemove$' -count=1","paths":["internal/artifact/domain/validation.go"]}]})

rec("003", 203, "ar-concurrency-lock-get-003", "diagnosis", "concurrency并发问题",
"""并发跑 Catalog 的 Get 和 Create 一直报 data race，贴一段：\n==================\nWARNING: DATA RACE\nRead at 0x00c00019ae40 by goroutine 8:\n  artifact-registry/internal/artifact/infrastructure/memory.(*Catalog).Get()\n      internal/artifact/infrastructure/memory/catalog.go:58\nPrevious write at 0x00c00019ae40 by goroutine 9:\n先别改代码，帮我查清楚哪里没锁住。""",
"""go test -race ./internal/artifact/infrastructure/memory -run '^TestGetDoesNotAlias$' -count=1
go test -race ./internal/artifact/infrastructure/memory -run '^TestGetBlobDoesNotAlias$' -count=1
go test -race ./internal/artifact/infrastructure/memory -run '^TestGetTagDoesNotAlias$' -count=1
go test -race ./internal/artifact/infrastructure/memory -run '^TestGetUploadDoesNotAlias$' -count=1
go test -race ./internal/artifact/infrastructure/memory -run '^TestCatalogConcurrentAccess$' -count=1""",
"""文件: internal/artifact/infrastructure/memory/catalog.go, internal/artifact/infrastructure/memory/catalog_more.go 符号: Catalog.Get, Catalog.GetBlob, Catalog.GetTag, Catalog.GetUpload 机制: Get 类读方法不加读锁直接读内部 map 并返回内部指针，并发写入与读取共享 map 和底层引用产生竞态和引用逃逸""",
"""1. -race 跑并发用例必须稳定报 DATA RACE 并指向 Catalog.Get 读路径；Get/GetBlob/GetTag/GetUpload 返回的也必须是副本而不是内部指针；2. 全程不得改任何代码文件，产生零代码差异；3. 结论要落到具体文件和缺失锁的读路径；4. 讲清楚为什么并发时才会现竞态。""",
{"version":1,"claims":[
 {"id":"get_no_alias","description":"Get 不返回内部指针","test":"TestGetDoesNotAlias","command":"go test -race ./internal/artifact/infrastructure/memory -run '^TestGetDoesNotAlias$' -count=1","paths":["internal/artifact/infrastructure/memory/catalog.go"]},
 {"id":"get_blob_no_alias","description":"GetBlob 不返回内部指针","test":"TestGetBlobDoesNotAlias","command":"go test -race ./internal/artifact/infrastructure/memory -run '^TestGetBlobDoesNotAlias$' -count=1","paths":["internal/artifact/infrastructure/memory/catalog.go"]},
 {"id":"get_tag_no_alias","description":"GetTag 不返回内部指针","test":"TestGetTagDoesNotAlias","command":"go test -race ./internal/artifact/infrastructure/memory -run '^TestGetTagDoesNotAlias$' -count=1","paths":["internal/artifact/infrastructure/memory/catalog_more.go"]},
 {"id":"get_upload_no_alias","description":"GetUpload 不返回内部指针","test":"TestGetUploadDoesNotAlias","command":"go test -race ./internal/artifact/infrastructure/memory -run '^TestGetUploadDoesNotAlias$' -count=1","paths":["internal/artifact/infrastructure/memory/catalog_more.go"]},
 {"id":"catalog_concurrent_access","description":"并发 Get/Create 稳定报竞态","test":"TestCatalogConcurrentAccess","command":"go test -race ./internal/artifact/infrastructure/memory -run '^TestCatalogConcurrentAccess$' -count=1","paths":["internal/artifact/infrastructure/memory/catalog.go"]}]})

rec("004", 204, "ar-context-timeout-middleware-004", "diagnosis", "context相关问题",
"""超时中间件好像根本没生效，慢请求不会被取消，租户中间件也没把 tenant 传下去。文件先不要改，帮我查清楚是哪里的问题。""",
"""go test ./internal/distribution/transport/http -run '^TestTimeoutMiddlewareCancels$' -count=1
go test ./internal/distribution/transport/http -run '^TestTenantMiddlewarePropagates$' -count=1
go test ./internal/distribution/transport/http -run '^TestWaitForReadyHonorsCancel$' -count=1
go test ./internal/distribution/transport/http -run '^TestActorFromContext$' -count=1
go test ./internal/distribution/transport/http -run '^TestRequestActor$' -count=1
go test ./internal/distribution/transport/http -run '^TestMiddlewareCtx$' -count=1""",
"""文件: internal/distribution/transport/http/middleware.go, internal/distribution/transport/http/management.go 符号: TimeoutMiddleware, TenantMiddleware, waitForReady, actorFromContext 机制: 中间件用 context.Background 而非请求 ctx 创建超时/传值，取消传播与租户信息在中间件层被切断""",
"""1. 慢请求在超时后要被取消，tenant 要能从 ctx 取到，已取消的 ctx 要让 waitForReady 立刻返回；2. 全程不得改任何代码文件，产生零代码差异；3. 结论要落到具体文件和把请求 ctx 换成 background 的那几处；4. 讲清楚为什么超时和租户都没传下去。""",
{"version":1,"claims":[
 {"id":"timeout_middleware_cancels","description":"TimeoutMiddleware 取消慢请求","test":"TestTimeoutMiddlewareCancels","command":"go test ./internal/distribution/transport/http -run '^TestTimeoutMiddlewareCancels$' -count=1","paths":["internal/distribution/transport/http/middleware.go"]},
 {"id":"tenant_middleware_propagates","description":"TenantMiddleware 传递 tenant","test":"TestTenantMiddlewarePropagates","command":"go test ./internal/distribution/transport/http -run '^TestTenantMiddlewarePropagates$' -count=1","paths":["internal/distribution/transport/http/middleware.go"]},
 {"id":"wait_ready_honors_cancel","description":"waitForReady 尊重取消","test":"TestWaitForReadyHonorsCancel","command":"go test ./internal/distribution/transport/http -run '^TestWaitForReadyHonorsCancel$' -count=1","paths":["internal/distribution/transport/http/management.go"]},
 {"id":"actor_from_context","description":"actorFromContext 读取 ctx","test":"TestActorFromContext","command":"go test ./internal/distribution/transport/http -run '^TestActorFromContext$' -count=1","paths":["internal/distribution/transport/http/management.go"]},
 {"id":"request_actor","description":"requestActor 读取请求 ctx","test":"TestRequestActor","command":"go test ./internal/distribution/transport/http -run '^TestRequestActor$' -count=1","paths":["internal/distribution/transport/http/management.go"]},
 {"id":"middleware_ctx","description":"middlewareCtx 返回请求 ctx","test":"TestMiddlewareCtx","command":"go test ./internal/distribution/transport/http -run '^TestMiddlewareCtx$' -count=1","paths":["internal/distribution/transport/http/middleware.go"]}]})

rec("005", 205, "ar-slice-histogram-alias-005", "bugfix", "slice相关问题",
"""Histogram 的 Values 拿完以后再 Quantile 会把结果冲掉，Span 的 AttributesCopy 改一份把原属性也改了。你跑下测试就能看到，帮我修好。""",
"""go test ./internal/platform/observability -run '^TestHistogramValuesIsolated$' -count=1
go test ./internal/platform/observability -run '^TestHistogramQuantileCopy$' -count=1
go test ./internal/platform/observability -run '^TestSpanAttributesCopy$' -count=1
go test ./internal/platform/observability -run '^TestSpanAttributeKeys$' -count=1""",
"""文件: internal/platform/observability/metrics.go, internal/platform/observability/trace.go 符号: Histogram.Values, Histogram.Quantile, Span.AttributesCopy, Span.AttributeKeys 机制: 返回共享底层数组/共享 map，Quantile 就地排序把已返回快照冲掉，AttributesCopy 返回共享 map 被调用方改动污染""",
"""1. Values 返回的快照不能被后续 Quantile 改动，AttributesCopy 改动不影响原属性，AttributeKeys 要排序；2. 修好后这四条连续 20 遍全绿；3. 撤回任一副本或排序，对应测试立刻又红；4. 全量测试无回归，只验证公开行为。""",
{"version":1,"claims":[
 {"id":"histogram_values_isolated","description":"Values 快照不被 Quantile 冲掉","test":"TestHistogramValuesIsolated","command":"go test ./internal/platform/observability -run '^TestHistogramValuesIsolated$' -count=1","paths":["internal/platform/observability/metrics.go"]},
 {"id":"histogram_quantile_copy","description":"Quantile 不改内部顺序","test":"TestHistogramQuantileCopy","command":"go test ./internal/platform/observability -run '^TestHistogramQuantileCopy$' -count=1","paths":["internal/platform/observability/metrics.go"]},
 {"id":"span_attributes_copy","description":"AttributesCopy 返回独立副本","test":"TestSpanAttributesCopy","command":"go test ./internal/platform/observability -run '^TestSpanAttributesCopy$' -count=1","paths":["internal/platform/observability/trace.go"]},
 {"id":"span_attribute_keys","description":"AttributeKeys 排序","test":"TestSpanAttributeKeys","command":"go test ./internal/platform/observability -run '^TestSpanAttributeKeys$' -count=1","paths":["internal/platform/observability/trace.go"]}]})

rec("006", 206, "ar-defer-swallow-scan-006", "diagnosis", "defer相关问题",
"""扫描引擎失败的时候不报错，合并不同 artifact 的报告也不报错。先别改代码，帮我查清楚原因。""",
"""go test ./internal/artifact/scanner -run '^TestScanReturnsError$' -count=1
go test ./internal/artifact/scanner -run '^TestScanAllReturnsError$' -count=1
go test ./internal/artifact/scanner -run '^TestMergeRejectsDifferentArtifacts$' -count=1
go test ./internal/artifact/scanner -run '^TestCombineRejectsDifferentArtifacts$' -count=1
go test ./internal/artifact/scanner -run '^TestValidateFindingsReturnsError$' -count=1""",
"""文件: internal/artifact/scanner/engine.go, internal/artifact/scanner/report.go 符号: Engine.Scan, Engine.ScanAll, Report.Merge, Report.Combine, Report.ValidateFindings 机制: 命名返回值配合 defer 把业务错误覆盖成 nil，扫描和校验失败被静默丢弃""",
"""1. 扫描失败要返回错误，不同 artifact 的 Merge/Combine 要报错，空 finding id 校验要报错；2. 全程不得改任何代码文件，产生零代码差异；3. 结论要落到具体文件和吞错的那段 defer；4. 讲清楚为什么这些错误都变成了 nil。""",
{"version":1,"claims":[
 {"id":"scan_returns_error","description":"Scan 返回扫描错误","test":"TestScanReturnsError","command":"go test ./internal/artifact/scanner -run '^TestScanReturnsError$' -count=1","paths":["internal/artifact/scanner/engine.go"]},
 {"id":"scan_all_returns_error","description":"ScanAll 返回首个错误","test":"TestScanAllReturnsError","command":"go test ./internal/artifact/scanner -run '^TestScanAllReturnsError$' -count=1","paths":["internal/artifact/scanner/engine.go"]},
 {"id":"merge_rejects_diff","description":"Merge 拒绝不同 artifact","test":"TestMergeRejectsDifferentArtifacts","command":"go test ./internal/artifact/scanner -run '^TestMergeRejectsDifferentArtifacts$' -count=1","paths":["internal/artifact/scanner/report.go"]},
 {"id":"combine_rejects_diff","description":"Combine 拒绝不同 artifact","test":"TestCombineRejectsDifferentArtifacts","command":"go test ./internal/artifact/scanner -run '^TestCombineRejectsDifferentArtifacts$' -count=1","paths":["internal/artifact/scanner/report.go"]},
 {"id":"validate_findings_error","description":"ValidateFindings 返回校验错误","test":"TestValidateFindingsReturnsError","command":"go test ./internal/artifact/scanner -run '^TestValidateFindingsReturnsError$' -count=1","paths":["internal/artifact/scanner/report.go"]}]})

rec("007", 207, "ar-concurrency-drain-replication-007", "bugfix", "concurrency并发问题",
"""复制协调器关掉之后还能往里塞任务直接 panic，Drain 数出来的任务数也不对，队列 Pop 之后再 Push 同一个 ID 就进不去了。帮我修好。""",
"""go test -race ./internal/artifact/replicationctl -run '^TestEnqueueAfterCloseReturnsError$' -count=1
go test -race ./internal/artifact/replicationctl -run '^TestDrainCountsTasks$' -count=1
go test -race ./internal/artifact/replicationctl -run '^TestQueuePopAllowsRepush$' -count=1
go test -race ./internal/artifact/replicationctl -run '^TestQueueLen$' -count=1
go test -race ./internal/artifact/replicationctl -run '^TestCoordinatorConcurrentDrain$' -count=1""",
"""文件: internal/artifact/replicationctl/coordinator.go, internal/artifact/replicationctl/queue.go 符号: Coordinator.Enqueue, Coordinator.Drain, Queue.Pop, Queue.Len 机制: 关闭后仍向已关闭 channel 发送导致 panic，Pop 不清理去重标记，Drain/Len 计数错位，channel 生命周期与去重状态不同步""",
"""1. Close 后 Enqueue 要返回错误而非 panic，Drain 要数对任务数，Pop 后能再 Push 同 ID，Len 要准确；2. 修好后这五条连续 20 遍全绿；3. 撤回任一修复，对应测试立刻又红；4. 全量测试无回归，只验证公开行为。""",
{"version":1,"claims":[
 {"id":"enqueue_after_close_errors","description":"Close 后 Enqueue 返回错误","test":"TestEnqueueAfterCloseReturnsError","command":"go test -race ./internal/artifact/replicationctl -run '^TestEnqueueAfterCloseReturnsError$' -count=1","paths":["internal/artifact/replicationctl/coordinator.go"]},
 {"id":"drain_counts_tasks","description":"Drain 数对任务数","test":"TestDrainCountsTasks","command":"go test -race ./internal/artifact/replicationctl -run '^TestDrainCountsTasks$' -count=1","paths":["internal/artifact/replicationctl/coordinator.go"]},
 {"id":"queue_pop_repush","description":"Pop 后可再 Push 同 ID","test":"TestQueuePopAllowsRepush","command":"go test -race ./internal/artifact/replicationctl -run '^TestQueuePopAllowsRepush$' -count=1","paths":["internal/artifact/replicationctl/queue.go"]},
 {"id":"queue_len","description":"Queue.Len 准确","test":"TestQueueLen","command":"go test -race ./internal/artifact/replicationctl -run '^TestQueueLen$' -count=1","paths":["internal/artifact/replicationctl/queue.go"]},
 {"id":"coordinator_concurrent_drain","description":"并发 Drain 不报竞态","test":"TestCoordinatorConcurrentDrain","command":"go test -race ./internal/artifact/replicationctl -run '^TestCoordinatorConcurrentDrain$' -count=1","paths":["internal/artifact/replicationctl/coordinator.go"]}]})

rec("008", 208, "ar-state-quota-ledger-008", "diagnosis", "其他问题",
"""quota 的 reservation 从 committed 又能转回 reserved，状态写进去读出来还是旧的，按状态数数量也多一个。文件先不要改，帮我查清楚原因。""",
"""go test ./internal/artifact/quota -run '^TestMeterAddOverLimit$' -count=1
go test ./internal/artifact/quota -run '^TestLedgerRejectsInvalidTransition$' -count=1
go test ./internal/artifact/quota -run '^TestLedgerPersistsState$' -count=1
go test ./internal/artifact/quota -run '^TestLedgerCountByState$' -count=1""",
"""文件: internal/artifact/quota/meter.go, internal/artifact/quota/ledger.go 符号: Meter.Add, Ledger.Put, Ledger.CountByState 机制: 状态机转换表把 reserved 目标当成无条件创建，转换后写回旧状态，越限判定与计数边界错位""",
"""1. 计量要正确处理越限边界，committed 不能转回 reserved，转换后要持久化新状态，按状态计数要准确；2. 全程不得改任何代码文件，产生零代码差异；3. 结论要落到具体文件和状态机三处错位；4. 讲清楚为什么状态和计数都不对。""",
{"version":1,"claims":[
 {"id":"meter_add_over_limit","description":"Meter.Add 正确判定越限","test":"TestMeterAddOverLimit","command":"go test ./internal/artifact/quota -run '^TestMeterAddOverLimit$' -count=1","paths":["internal/artifact/quota/meter.go"]},
 {"id":"ledger_rejects_invalid_transition","description":"Ledger 拒绝非法转换","test":"TestLedgerRejectsInvalidTransition","command":"go test ./internal/artifact/quota -run '^TestLedgerRejectsInvalidTransition$' -count=1","paths":["internal/artifact/quota/ledger.go"]},
 {"id":"ledger_persists_state","description":"Ledger 持久化新状态","test":"TestLedgerPersistsState","command":"go test ./internal/artifact/quota -run '^TestLedgerPersistsState$' -count=1","paths":["internal/artifact/quota/ledger.go"]},
 {"id":"ledger_count_by_state","description":"Ledger.CountByState 准确","test":"TestLedgerCountByState","command":"go test ./internal/artifact/quota -run '^TestLedgerCountByState$' -count=1","paths":["internal/artifact/quota/ledger.go"]}]})

rec("009", 209, "ar-slice-gc-compaction-009", "bugfix", "slice相关问题",
"""GC 的 Sweep 跑完以后 Retained 结果就乱了，MarkSet 的 Size 也多一个，Diff 数出来也不对。你直接跑下测试会挂，帮我修好。""",
"""go test ./internal/artifact/gc -run '^TestSweepIsRepeatable$' -count=1
go test ./internal/artifact/gc -run '^TestRetainedAfterSweep$' -count=1
go test ./internal/artifact/gc -run '^TestMarkSetSize$' -count=1
go test ./internal/artifact/gc -run '^TestSweeperDiff$' -count=1""",
"""文件: internal/artifact/gc/mark.go, internal/artifact/gc/sweep.go 符号: MarkSet.Size, Sweeper.Sweep, Sweeper.Retained, Sweeper.Diff 机制: Sweep/Retained 用 s.keys[:0] 原地压缩把内部 key 列表冲掉，Size/Diff 边界多算导致计数错位""",
"""1. Sweep 要可重复执行，Sweep 后 Retained 结果要正确，MarkSet.Size 要准确，Diff 要数对删除/保留数量；2. 修好后这四条连续 20 遍全绿；3. 撤回任一副本或计数修复，对应测试立刻又红；4. 全量测试无回归，只验证公开行为。""",
{"version":1,"claims":[
 {"id":"sweep_repeatable","description":"Sweep 可重复执行","test":"TestSweepIsRepeatable","command":"go test ./internal/artifact/gc -run '^TestSweepIsRepeatable$' -count=1","paths":["internal/artifact/gc/sweep.go"]},
 {"id":"retained_after_sweep","description":"Sweep 后 Retained 正确","test":"TestRetainedAfterSweep","command":"go test ./internal/artifact/gc -run '^TestRetainedAfterSweep$' -count=1","paths":["internal/artifact/gc/sweep.go"]},
 {"id":"mark_set_size","description":"MarkSet.Size 准确","test":"TestMarkSetSize","command":"go test ./internal/artifact/gc -run '^TestMarkSetSize$' -count=1","paths":["internal/artifact/gc/mark.go"]},
 {"id":"sweeper_diff","description":"Sweeper.Diff 数对数量","test":"TestSweeperDiff","command":"go test ./internal/artifact/gc -run '^TestSweeperDiff$' -count=1","paths":["internal/artifact/gc/sweep.go"]}]})

rec("010", 210, "ar-error-wrap-upload-010", "bugfix", "error异常错误",
"""上传 writer 和 session 报的错用 errors.Is 判不到 sentinel，重复 chunk、nil reader、过期 session 全都一样。帮我修掉。""",
"""go test ./internal/artifact/uploader -run '^TestWriterAddDuplicateChunk$' -count=1
go test ./internal/artifact/uploader -run '^TestWriterAddNilReader$' -count=1
go test ./internal/artifact/uploader -run '^TestSessionValidateExpired$' -count=1
go test ./internal/artifact/uploader -run '^TestSessionValidateOverlap$' -count=1
go test ./internal/artifact/uploader -run '^TestWriterPartReaderMissing$' -count=1
go test ./internal/artifact/uploader -run '^TestSessionEndOffsetEmpty$' -count=1
go test ./internal/artifact/uploader -run '^TestWriterRemoveMissing$' -count=1
go test ./internal/artifact/uploader -run '^TestSessionChunkAtMissing$' -count=1""",
"""文件: internal/artifact/uploader/writer.go, internal/artifact/uploader/session.go 符号: Writer.Add, Writer.PartReader, Writer.Remove, Session.Validate, Session.EndOffset, Session.ChunkAt 机制: 用 %v 而非 %w 包装 sentinel，errors.Is 无法识别 ErrDuplicateChunk/ErrNilReader/ErrExpiredSession 等错误链""",
"""1. 重复 chunk、nil reader、过期 session、缺失 chunk 的错误都要能被 errors.Is 认成对应 sentinel；2. 修好后这八条连续 20 遍全绿；3. 撤回任一 %w 包装，对应测试立刻又红；4. 全量测试无回归，只验证公开行为。""",
{"version":1,"claims":[
 {"id":"writer_dup_chunk","description":"Add 重复 chunk 包装 sentinel","test":"TestWriterAddDuplicateChunk","command":"go test ./internal/artifact/uploader -run '^TestWriterAddDuplicateChunk$' -count=1","paths":["internal/artifact/uploader/writer.go"]},
 {"id":"writer_nil_reader","description":"Add nil reader 包装 sentinel","test":"TestWriterAddNilReader","command":"go test ./internal/artifact/uploader -run '^TestWriterAddNilReader$' -count=1","paths":["internal/artifact/uploader/writer.go"]},
 {"id":"session_expired","description":"Validate 过期包装 sentinel","test":"TestSessionValidateExpired","command":"go test ./internal/artifact/uploader -run '^TestSessionValidateExpired$' -count=1","paths":["internal/artifact/uploader/session.go"]},
 {"id":"session_overlap","description":"Validate 重叠包装 sentinel","test":"TestSessionValidateOverlap","command":"go test ./internal/artifact/uploader -run '^TestSessionValidateOverlap$' -count=1","paths":["internal/artifact/uploader/session.go"]},
 {"id":"writer_part_reader","description":"PartReader 缺失包装 sentinel","test":"TestWriterPartReaderMissing","command":"go test ./internal/artifact/uploader -run '^TestWriterPartReaderMissing$' -count=1","paths":["internal/artifact/uploader/writer.go"]},
 {"id":"session_end_offset","description":"EndOffset 空 session 包装 sentinel","test":"TestSessionEndOffsetEmpty","command":"go test ./internal/artifact/uploader -run '^TestSessionEndOffsetEmpty$' -count=1","paths":["internal/artifact/uploader/session.go"]},
 {"id":"writer_remove","description":"Remove 缺失包装 sentinel","test":"TestWriterRemoveMissing","command":"go test ./internal/artifact/uploader -run '^TestWriterRemoveMissing$' -count=1","paths":["internal/artifact/uploader/writer.go"]},
 {"id":"session_chunk_at","description":"ChunkAt 缺失包装 sentinel","test":"TestSessionChunkAtMissing","command":"go test ./internal/artifact/uploader -run '^TestSessionChunkAtMissing$' -count=1","paths":["internal/artifact/uploader/session.go"]}]})

for k,v in R.items():
    (OUT/f'collection_{k}.json').write_text(json.dumps(v, ensure_ascii=False, indent=2)+'\n')
print("wrote", len(R), "files")
