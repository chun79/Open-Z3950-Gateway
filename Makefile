.PHONY: test lint build ci-local

# 运行所有测试
test:
	@echo "Running unit and integration tests..."
	go test -v ./pkg/... ./cmd/...

# 运行静态代码检查 (需要先安装 golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# 编译验证
build:
	@echo "Verifying build..."
	go mod tidy
	go build -o /dev/null cmd/gateway/main.go
	go build -o /dev/null cmd/zserver/main.go

# 本地 CI 流水线一键运行
ci-local: build test
	@echo "Local CI pipeline PASSED!"
