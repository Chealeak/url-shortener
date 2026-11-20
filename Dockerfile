FROM golang:1.26.4-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/url-shortener ./cmd/url-shortener

FROM alpine:3.22

RUN adduser -D -H -u 10001 app

WORKDIR /app

COPY --from=build /out/url-shortener /app/url-shortener
COPY config/ /app/config/

RUN touch /app/.env && chown -R app:app /app

USER app

ENV CONFIG_PATH=/app/config/local.yaml

EXPOSE 8082

ENTRYPOINT ["/app/url-shortener"]
