# syntax=docker/dockerfile:1
FROM golang:1.25-alpine
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && go env -w GOPROXY=https://goproxy.cn,direct \
    && go env -w GOTOOLCHAIN=local
ENV TZ=Asia/Shanghai CGO_ENABLED=0 GOOS=linux GOPROXY=https://goproxy.cn,direct
WORKDIR /work
# Image is invoked by the control plane with `go build ...`
CMD ["go", "version"]
