FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 65532 app \
    && mkdir -p /data \
    && chown app:app /data
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /src/project/web /app/web
COPY --from=build /src/project/openapi.yaml /app/openapi.yaml
ENV HTTP_ADDR=:8080 \
    WEB_DIR=/app/web \
    OPENAPI_PATH=/app/openapi.yaml \
    DB_DSN=file:/data/board.db?_pragma=busy_timeout(5000)
USER app
EXPOSE 8080
ENTRYPOINT ["/app/api"]
