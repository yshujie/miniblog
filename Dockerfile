# 使用官方 Golang 镜像作为基础镜像
FROM golang:1.24

# 设置工作目录为 /app
WORKDIR /app

# 设置代理
ENV GOPROXY=https://goproxy.cn,direct

# 将本地 go.mod 和 go.sum 复制到容器中
COPY go.mod go.sum ./

# 安装依赖
RUN go mod tidy

# 将整个项目目录复制到容器中
COPY . .

# 安装 air 工具，用于自动热重载
RUN go install github.com/cosmtrek/air@latest

# 设置容器启动时运行 air
CMD ["air"]
