# Simple Makefile for drill

.PHONY: watch
watch:
	watchexec -e go --wrap-process session --on-busy-update=restart --clear -- "go run cmd/drill/main.go; rm -f __debug_bin*"

.PHONY: fmt
fmt:
	go fmt ./...
