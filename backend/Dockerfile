FROM golang:1.25-alpine AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates git
ENV GOPROXY=https://goproxy.cn,direct \
    GOTOOLCHAIN=auto \
    CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/raftnode ./cmd/raftnode \
    && go build -trimpath -ldflags="-s -w" -o /out/raftctl ./cmd/raftctl

FROM alpine:3.21
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo Asia/Shanghai > /etc/timezone
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /out/raftnode /usr/local/bin/raftnode
COPY --from=builder /out/raftctl /usr/local/bin/raftctl
EXPOSE 8080
ENTRYPOINT ["raftnode"]
