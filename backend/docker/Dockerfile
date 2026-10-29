# syntax=docker/dockerfile:1

# Stage 1: build
FROM golang:1.22-alpine AS build
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/huginn-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/huginn-worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/huginn-cli ./cmd/cli

# Stage 2: runtime
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates wget && \
    addgroup -S huginn && adduser -S -G huginn huginn

WORKDIR /app
COPY --from=build /out/huginn-api /out/huginn-worker /out/huginn-cli /app/
COPY migrations /app/migrations

RUN chown -R huginn:huginn /app && chmod +x /app/huginn-*

USER huginn
EXPOSE 8080

ENTRYPOINT ["/app/huginn-api"]
