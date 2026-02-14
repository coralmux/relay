FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o coralmux-relay ./cmd/relay/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/coralmux-relay /usr/local/bin/
EXPOSE 8443
ENTRYPOINT ["coralmux-relay"]
