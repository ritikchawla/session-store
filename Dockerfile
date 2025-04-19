FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o session-store

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/session-store .
EXPOSE 8080
CMD ["./session-store"]