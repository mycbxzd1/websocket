# 使用官方 Golang 镜像作为构建环境
FROM golang:1.23-alpine as builder

# 设置工作目录
WORKDIR /go/src/app

# 将当前目录内容复制到容器中的工作目录
COPY . .

# 下载项目的依赖包
RUN go mod tidy

# 编译 Go 项目为静态二进制文件
RUN go build -o my-go-project .

# 使用较小的 Alpine 镜像作为运行时环境
FROM alpine:latest

# 安装必要的运行时依赖库 (如果需要的话)
RUN apk --no-cache add ca-certificates

# 设置工作目录
WORKDIR /root/

# 从 builder 镜像复制编译后的二进制文件到当前镜像
COPY --from=builder /go/src/app/my-go-project .

# 暴露服务所需的端口
EXPOSE 8080

# 启动二进制文件
CMD ["./my-go-project"]
