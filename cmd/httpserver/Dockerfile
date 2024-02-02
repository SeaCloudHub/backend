FROM golang:1.21-alpine3.18 as builder

RUN apk update && apk add --no-cache git make ca-certificates tzdata openssh
WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go mod verify

RUN go build -ldflags "-s -w" -o server ./cmd/httpserver/main.go

FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime
RUN adduser -D -g '' appuser

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/server /server

USER appuser

CMD ["/server"]