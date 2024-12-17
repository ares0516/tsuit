# Makefile

# 二进制文件名称
BINARY_CLIENT = yamux-client
BINARY_SERVER = yamux-server

# 模块名称（与 go.mod 中的一致）
MODULE_NAME = your_module_name

# 默认目标
.PHONY: all build build-client build-server clean

all: build

# 构建所有目标
build: build-client build-server

# 构建客户端
build-client:
    @echo "正在构建客户端..."
    go build -o $(BINARY_CLIENT) $(MODULE_NAME)/client
    @echo "客户端构建完成: $(BINARY_CLIENT)"

# 构建服务端
build-server:
    @echo "正在构建服务端..."
    go build -o $(BINARY_SERVER) $(MODULE_NAME)/server
    @echo "服务端构建完成: $(BINARY_SERVER)"

# 清理生成的二进制文件
clean:
    @echo "正在清理二进制文件..."
    rm -f $(BINARY_CLIENT) $(BINARY_SERVER)
    @echo "清理完成。"