FROM golang:1.22-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE=1"
COPY . .
RUN CGO_ENABLED=1 go build -ldflags '-extldflags "-static" -w -s' -o /app ./cmd/dashboard/main.go

FROM alpine AS certs
RUN apk update && apk add ca-certificates

FROM busybox:stable-musl

ARG TARGETOS
ARG TARGETARCH

COPY --from=certs /etc/ssl/certs /etc/ssl/certs
COPY ./script/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /dashboard
# COPY dist/dashboard-${TARGETOS}-${TARGETARCH} ./app
COPY resource ./resource
COPY --from=builder ./app ./app
VOLUME ["/dashboard/data"]
EXPOSE 80 5555
ARG TZ=Asia/Shanghai
ENV TZ=$TZ
ENTRYPOINT ["/entrypoint.sh"]
