# kubeboard 开发与测试入口
.PHONY: help build run test test-backend test-frontend test-race vet fmt clean

help:
	@echo "make build         构建后端二进制（bin/kubeboard）"
	@echo "make test          跑全部测试（后端 + 前端）"
	@echo "make test-backend  仅后端 go test"
	@echo "make test-frontend 仅前端 YAML 转换测试"
	@echo "make test-race     后端测试 + 数据竞争检测"
	@echo "make vet           go vet 静态检查"
	@echo "make build-web     构建前端产物（web/dist）"
	@echo "make run           本地启动（需 kubeconfig）"

build:
	go build -o bin/kubeboard ./cmd/kubeboard

build-web:
	cd web && npm run build

test: test-backend test-frontend

test-backend:
	go test ./...

test-frontend:
	cd web && npm test

test-race:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './web/*')

clean:
	rm -rf bin
