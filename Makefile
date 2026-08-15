.PHONY: build install clean build-all test

QG_APP_NAME = qg
VERSION = 1.0.0
BUILD_DIR = bin
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin

# 编译当前平台
build:
	mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(QG_APP_NAME) ./cmd

# 安装到系统
install: build
	mkdir -p $(BINDIR)
	cp $(BUILD_DIR)/$(QG_APP_NAME) $(BINDIR)/

# 清理编译产物
clean:
	rm -rf $(BUILD_DIR)

# 运行测试
test:
	go test -v ./...

# 交叉编译所有平台
build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(QG_APP_NAME)-darwin-amd64 ./cmd
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(QG_APP_NAME)-darwin-arm64 ./cmd
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(QG_APP_NAME)-linux-amd64 ./cmd
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(QG_APP_NAME)-linux-arm64 ./cmd
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(QG_APP_NAME)-windows-amd64.exe ./cmd

# 打包发布
release: build-all
	cd $(BUILD_DIR) && \
	tar -czvf $(QG_APP_NAME)-darwin-amd64.tar.gz $(QG_APP_NAME)-darwin-amd64 && \
	tar -czvf $(QG_APP_NAME)-darwin-arm64.tar.gz $(QG_APP_NAME)-darwin-arm64 && \
	tar -czvf $(QG_APP_NAME)-linux-amd64.tar.gz $(QG_APP_NAME)-linux-amd64 && \
	tar -czvf $(QG_APP_NAME)-linux-arm64.tar.gz $(QG_APP_NAME)-linux-arm64 && \
	zip $(QG_APP_NAME)-windows-amd64.zip $(QG_APP_NAME)-windows-amd64.exe

# 开发模式运行
dev:
	go run ./cmd
