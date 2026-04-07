.PHONY: build install clean build-all test

APP_NAME = cmt
VERSION = 1.0.0
BUILD_DIR = bin

# 编译当前平台
build:
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/cmt

# 安装到系统
install: build
	cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/

# 清理编译产物
clean:
	rm -rf $(BUILD_DIR)

# 运行测试
test:
	go test -v ./...

# 交叉编译所有平台
build-all:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/cmt
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/cmt
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/cmt
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/cmt
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/cmt

# 打包发布
release: build-all
	cd $(BUILD_DIR) && \
	tar -czvf $(APP_NAME)-darwin-amd64.tar.gz $(APP_NAME)-darwin-amd64 && \
	tar -czvf $(APP_NAME)-darwin-arm64.tar.gz $(APP_NAME)-darwin-arm64 && \
	tar -czvf $(APP_NAME)-linux-amd64.tar.gz $(APP_NAME)-linux-amd64 && \
	tar -czvf $(APP_NAME)-linux-arm64.tar.gz $(APP_NAME)-linux-arm64 && \
	zip $(APP_NAME)-windows-amd64.zip $(APP_NAME)-windows-amd64.exe

# 开发模式运行
dev:
	go run ./cmd/cmt
