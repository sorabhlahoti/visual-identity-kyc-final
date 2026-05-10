FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY . ./
ARG TARGET=server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/${TARGET}

FROM alpine:3.20
RUN apk add --no-cache su-exec \
    && adduser -D -H appuser \
    && mkdir -p /data /app \
    && chown -R appuser:appuser /data /app
WORKDIR /app
COPY --from=builder /out/app /app/app
COPY openapi.yaml /app/openapi.yaml
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh \
    && chown appuser:appuser /app/app /app/openapi.yaml
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
