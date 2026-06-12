FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum* ./
RUN if [ -f go.sum ]; then go mod download; fi

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /out/ethercat-simulator .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /out/ethercat-simulator .

RUN chown -R appuser:appgroup /app
USER appuser

ENV API_ENDPOINT=http://backend:8080
ENV INTERVAL_SECONDS=7200
ENV DEVICE_COUNT=50
ENV HIGH_SCALE_MODE=false
ENV BACKFILL_DAYS=7

CMD ["./ethercat-simulator"]
