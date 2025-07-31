FROM golang:1.24.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG APP_ENTRY=./cmd/api
ARG OUTPUT_NAME=main

RUN CGO_ENABLED=0 GOOS=linux go build -o ${OUTPUT_NAME} ${APP_ENTRY}

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

ARG OUTPUT_NAME=main
COPY --from=builder /app/${OUTPUT_NAME} .

RUN chmod +x /app/${OUTPUT_NAME}

EXPOSE 8080

CMD ["./main"]
