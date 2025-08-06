FROM golang:1.24.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o memdb ./cmd/database


FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/main /app/memdb ./

RUN chmod +x /app/main /app/memdb

EXPOSE 8080

CMD ["./main"]
