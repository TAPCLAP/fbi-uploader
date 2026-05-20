FROM docker.io/golang:1.25 AS builder
ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod ./
RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -trimpath -o /out/fbi-uploader ./cmd/fbi-uploader && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -trimpath -o /out/fbi-app-token ./cmd/fbi-app-token

FROM docker.io/alpine:3.23.4 AS certificates
RUN apk add --no-cache ca-certificates

FROM scratch

COPY --from=certificates /etc/ssl /etc/ssl
COPY --from=builder /out/fbi-uploader /fbi-uploader

ENTRYPOINT ["/fbi-uploader"]
