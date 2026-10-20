FROM node:24-bookworm-slim AS frontend
WORKDIR /ui
COPY ["智慧养老手表管理系统/oldIot/package.json", "智慧养老手表管理系统/oldIot/package-lock.json", "./"]
RUN npm ci --no-audit --no-fund
COPY ["智慧养老手表管理系统/oldIot/", "./"]
RUN npm run test && npm run typecheck && npm run build

FROM golang:1.27.2-bookworm AS backend
WORKDIR /src
COPY ["智慧养老手表管理系统/backend/go.mod", "智慧养老手表管理系统/backend/go.sum", "./"]
RUN go mod download
COPY ["智慧养老手表管理系统/backend/", "./"]
COPY --from=frontend /ui/dist/ internal/handlers/web/
RUN CGO_ENABLED=0 go build -trimpath -o /out/suining ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/healthcheck ./cmd/healthcheck \
 && CGO_ENABLED=0 go build -trimpath -o /out/protect-notes ./cmd/protect-notes

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/* \
 && groupadd -g 10001 suining && useradd -r -u 10001 -g suining suining \
 && mkdir -p /app /data/images && chown -R suining:suining /app /data
COPY --from=backend /out/ /app/
USER 10001:10001
WORKDIR /app
ENV ADDR=0.0.0.0:8080 IMAGE_DIR=/data/images SESSION_BACKEND=database
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=4s --start-period=40s CMD ["/app/healthcheck"]
ENTRYPOINT ["/app/suining"]
