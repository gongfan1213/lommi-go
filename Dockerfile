# 多阶段构建Dockerfile

# 第一阶段：构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o loomi-go ./cmd/loomi/

# 第二阶段：运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1001 -S loomi && \
    adduser -u 1001 -S loomi -G loomi

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/loomi-go .

# 复制配置文件
COPY --from=builder /app/config ./config
COPY --from=builder /app/test_config.yaml .

# 复制启动脚本
COPY --from=builder /app/run_tests.sh .
RUN chmod +x run_tests.sh

# 创建必要的目录
RUN mkdir -p logs uploads coverage test-reports && \
    chown -R loomi:loomi /app

# 切换到非root用户
USER loomi

# 暴露端口
EXPOSE 8080 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动命令
CMD ["./loomi-go"]
