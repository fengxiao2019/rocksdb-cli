#!/bin/bash

# RocksDB CLI Docker 构建脚本
# 使用可选的代理设置构建 Docker 镜像

set -e

echo "🐳 构建 RocksDB CLI Docker 镜像..."

# 检查是否需要代理设置
PROXY_ARGS=""
if [ -n "$HTTP_PROXY" ] || [ -n "$HTTPS_PROXY" ]; then
    echo "📡 检测到代理设置，使用代理构建"
    PROXY_ARGS="--build-arg HTTP_PROXY=$HTTP_PROXY --build-arg HTTPS_PROXY=$HTTPS_PROXY --build-arg http_proxy=$HTTP_PROXY --build-arg https_proxy=$HTTPS_PROXY"
else
    echo "🌐 无代理设置，直接构建"
fi

# 构建 Docker 镜像
docker build $PROXY_ARGS -t rocksdb-cli .

if [ $? -eq 0 ]; then
    echo "✅ Docker 镜像构建成功!"
    echo ""
    echo "测试命令:"
    echo "  docker run --rm rocksdb-cli --help"
    echo "  docker run -it --rm -v \"/path/to/db:/data\" rocksdb-cli --db /data"
else
    echo "❌ Docker 镜像构建失败"
    exit 1
fi 