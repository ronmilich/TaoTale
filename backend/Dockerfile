FROM node:24.19.0-bookworm-slim AS frontend
WORKDIR /src
RUN npm install -g pnpm@11.25.0
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
COPY frontend/tools/contract/package.json ./tools/contract/package.json
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM golang:1.27.4 AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/taotale ./cmd/taotale
RUN mkdir -m 0700 /out/media

FROM scratch
COPY --from=build /out/taotale /taotale
COPY --from=build --chown=10001:10001 /out/media /var/lib/taotale/media/
COPY --from=frontend /src/dist/client /opt/taotale/frontend/
ENV SERVE_FRONTEND_DIST=/opt/taotale/frontend
USER 10001:10001
ENTRYPOINT ["/taotale"]
