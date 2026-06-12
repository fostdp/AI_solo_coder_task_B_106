FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.Version=2.0.0" \
    -o /out/stone-relic-monitor .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /out/stone-relic-monitor .
COPY config.yaml .

RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8080 6060

HEALTHCHECK --interval=15s --timeout=5s --retries=5 \
    CMD wget --spider -q http://localhost:8080/health || exit 1

CMD ["./stone-relic-monitor"]
