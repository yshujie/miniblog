


# 构建阶段
FROM golang:1.21 AS builder

# 设置工作目录
WORKDIR /app

# 设置代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# 复制依赖文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
# CGO_ENABLED=0 禁用 CGO
# -ldflags="-w -s" 减小二进制文件大小
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/server/main.go

# 运行阶段
FROM alpine:latest

# 安装 CA 证书
RUN apk --no-cache add ca-certificates

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的应用
COPY --from=builder /app/main .
# 复制配置文件（根据需要调整）
COPY --from=builder /app/configs ./configs

# 暴露端口（根据你的应用需要修改）
EXPOSE 8080

# 运行应用
CMD ["./main"]