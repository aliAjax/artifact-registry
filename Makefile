GO ?= go
.PHONY: fmt test vet run lines
fmt:
	$(GO) fmt ./...
test:
	$(GO) test ./...
vet:
	$(GO) vet ./...
run:
	$(GO) run ./cmd/registry
lines:
	find . -name '*.go' -not -name '*_test.go' -print0 | xargs -0 wc -l
