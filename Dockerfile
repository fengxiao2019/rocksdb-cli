# RocksDB CLI Docker 构建文件
# 编译 RocksDB v10.2.1 以兼容 grocksdb v1.10.1
# 
# 构建命令:
#   docker build -t rocksdb-cli .
# 
# 运行命令:
#   docker run -it --rm -v "/path/to/db:/data" rocksdb-cli --db /data

FROM golang:1.23-bullseye AS builder

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG http_proxy
ARG https_proxy
ARG no_proxy

ENV DEBIAN_FRONTEND=noninteractive

# 安装 RocksDB 编译依赖
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    libgflags-dev \
    libsnappy-dev \
    zlib1g-dev \
    libbz2-dev \
    liblz4-dev \
    libzstd-dev \
    && rm -rf /var/lib/apt/lists/*

# 编译 RocksDB v10.2.1 (限制并行数以避免内存不足)
WORKDIR /tmp
RUN git clone https://github.com/facebook/rocksdb.git && \
    cd rocksdb && \
    git checkout v10.2.1 && \
    make shared_lib -j2 && \
    make install-shared && \
    ldconfig

# 切换到应用构建
WORKDIR /workspace

# 下载 Go 依赖
COPY go.mod go.sum ./
RUN go mod download

# 编译应用
COPY . .
ENV CGO_ENABLED=1
ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
RUN go build -ldflags "-X main.version=docker-v10.2.1" -o rocksdb-cli ./cmd

# 运行时镜像
FROM debian:bullseye-slim

ENV DEBIAN_FRONTEND=noninteractive

# 安装运行时依赖
RUN apt-get update && apt-get install -y \
    libgflags2.2 \
    libsnappy1v5 \
    liblz4-1 \
    libzstd1 \
    libbz2-1.0 \
    zlib1g \
    libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

# 复制 RocksDB 库和应用
COPY --from=builder /usr/local/lib/librocksdb* /usr/local/lib/
COPY --from=builder /workspace/rocksdb-cli /usr/local/bin/rocksdb-cli
RUN ldconfig && chmod +x /usr/local/bin/rocksdb-cli

# 创建非 root 用户
RUN useradd -r -s /bin/false -d /data rocksdb && \
    mkdir -p /data && chown rocksdb:rocksdb /data

USER rocksdb
WORKDIR /data

ENTRYPOINT ["/usr/local/bin/rocksdb-cli"] 