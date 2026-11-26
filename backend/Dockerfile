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

FROM scratch AS standard
COPY --from=build /out/taotale /taotale
COPY --from=build --chown=10001:10001 /out/media /var/lib/taotale/media/
COPY --from=frontend /src/dist/client /opt/taotale/frontend/
ENV SERVE_FRONTEND_DIST=/opt/taotale/frontend
USER 10001:10001
ENTRYPOINT ["/taotale"]

FROM golang:1.27.4-bookworm AS speech-worker
RUN curl -fsSL https://csspeechstorage.blob.core.windows.net/drop/1.53.0/SpeechSDK-Linux-1.53.0.tar.gz -o /tmp/speech.tgz \
    && echo '9f88b77c003c7dabe08623e7c4b41ce3f3bad109c79bcfea62f5d4a32861bc6f  /tmp/speech.tgz' | sha256sum -c - \
    && mkdir /opt/speech && tar xzf /tmp/speech.tgz --strip-components=1 -C /opt/speech && rm /tmp/speech.tgz
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ENV CGO_ENABLED=1 CGO_CFLAGS=-I/opt/speech/include/c_api CGO_LDFLAGS="-L/opt/speech/lib/x64 -lMicrosoft.CognitiveServices.Speech.core" LD_LIBRARY_PATH=/opt/speech/lib/x64
RUN go test -tags azure_sdk ./cmd/taotale-azure \
    && go build -tags azure_sdk -trimpath -o /taotale-azure ./cmd/taotale-azure

FROM debian:bookworm-slim AS speech-runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libasound2 libssl3 libstdc++6 libuuid1 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=speech-worker /opt/speech/lib/x64/ /opt/speech/lib/x64/
COPY --from=speech-worker /taotale-azure /usr/local/bin/taotale-azure
COPY --from=standard /taotale /taotale
COPY --from=standard --chown=10001:10001 /var/lib/taotale/media/ /var/lib/taotale/media/
COPY --from=standard /opt/taotale/frontend/ /opt/taotale/frontend/
ENV LD_LIBRARY_PATH=/opt/speech/lib/x64 SERVE_FRONTEND_DIST=/opt/taotale/frontend
USER 10001:10001
ENTRYPOINT ["/taotale"]

FROM standard AS default-runtime
