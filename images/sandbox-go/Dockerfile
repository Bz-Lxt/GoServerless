# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS agent
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates \
    && go env -w GOPROXY=https://goproxy.cn,direct
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /sandbox-agent ./cmd/sandbox-agent

FROM alpine:3.20
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && addgroup -g 65534 -S nobody || true \
    && adduser -S -D -H -u 65534 -G nobody nobody || true
ENV TZ=Asia/Shanghai
COPY --from=agent /sandbox-agent /usr/local/bin/sandbox-agent
USER 65534:65534
ENTRYPOINT ["/usr/local/bin/sandbox-agent"]
