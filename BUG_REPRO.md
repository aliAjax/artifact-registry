# BUG_REPRO：上传 writer/session 错误链断裂

## Bug 是什么
Writer/Session 用 %v 包装 sentinel，errors.Is 无法识别 ErrDuplicateChunk/ErrNilReader/ErrExpiredSession 等。

## 如何触发
```sh
go test ./internal/uploader -run '^TestWriterAddDuplicateChunk$' -count=1
go test ./internal/uploader -run '^TestWriterAddNilReader$' -count=1
go test ./internal/uploader -run '^TestSessionValidateExpired$' -count=1
go test ./internal/uploader -run '^TestSessionValidateOverlap$' -count=1
go test ./internal/uploader -run '^TestWriterPartReaderMissing$' -count=1
go test ./internal/uploader -run '^TestSessionEndOffsetEmpty$' -count=1
go test ./internal/uploader -run '^TestWriterRemoveMissing$' -count=1
go test ./internal/uploader -run '^TestSessionChunkAtMissing$' -count=1
```

## 错误信息
- expected ErrDuplicateChunk, got chunk already present: offset 0
