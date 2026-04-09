# Stage 1: Build
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

# 启用自动工具链下载，让 Go 1.24 基础镜像自动获取 go.mod 要求的 Go 1.26+
ENV GOTOOLCHAIN=auto

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -trimpath \
    -o site-health-monitor .

# Stage 2: CA 证书 + 创建数据目录
FROM alpine:latest AS certs
RUN apk --no-cache add ca-certificates && mkdir -p /data

# Stage 3: scratch (ultra-minimal)
FROM scratch

WORKDIR /

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=certs /data /data
COPY --from=builder /src/site-health-monitor /site-health-monitor
COPY --from=builder /src/templates /templates

# 配置数据持久化目录
# Docker 自动创建匿名 volume，容器 restart/stop/start 时数据不丢失
# docker rm -v 时会同步删除 volume，彻底清理
VOLUME /data

EXPOSE 9099

ENTRYPOINT ["/site-health-monitor"]
