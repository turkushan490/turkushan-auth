# 1. Frontend
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2. Go binary (pure Go SQLite driver, so no CGO and a fully static binary)
FROM golang:1-alpine AS build
WORKDIR /src
COPY . .
COPY --from=web /web/dist ./web/dist
RUN go mod tidy \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/turkushan-auth ./cmd/turkushan-auth \
 && mkdir -p /out/data

# 3. Runtime: no shell, no package manager. Starts as root only to chown /data,
#    then drops to PUID:PGID (default 99:100 = nobody:users on Unraid).
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/turkushan-auth /turkushan-auth
COPY --from=build /out/data /data
ENV DATA_DIR=/data \
    PORT=3010
EXPOSE 3010
VOLUME /data
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/turkushan-auth", "healthcheck"]
ENTRYPOINT ["/turkushan-auth"]
