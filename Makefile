.PHONY: test lint build ci-local frontend-install frontend-build frontend-copy

# 运行所有测试
test:
	@echo "Running unit and integration tests..."
	go test -v ./pkg/... ./cmd/gateway/...

# 运行静态代码检查 (需要先安装 golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# 前端依赖安装
frontend-install:
	@echo "Installing frontend dependencies..."
	cd webapp && npm install

# 前端构建
frontend-build:
	@echo "Building frontend..."
	cd webapp && npm run build

# 同步前端产物到 Go 包
frontend-copy:
	@echo "Copying frontend assets to pkg/ui/dist..."
	rm -rf pkg/ui/dist
	cp -r webapp/dist pkg/ui/dist

# 编译验证 (后端)
build-backend:
	@echo "Verifying backend build..."
	go mod tidy
	go build -o /dev/null cmd/gateway/main.go

# 全量编译验证 (前端 + 后端)
build: frontend-install frontend-build frontend-copy build-backend
	@echo "Full build verification PASSED!"

# 本地 CI 流水线一键运行
ci-local: build test
	@echo "Local CI pipeline PASSED!"