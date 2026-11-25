FROM golang:1.27.4 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/taotale ./cmd/taotale
RUN mkdir -m 0700 /out/media

FROM scratch
COPY --from=build /out/taotale /taotale
COPY --from=build --chown=10001:10001 /out/media /var/lib/taotale/media/
USER 10001:10001
ENTRYPOINT ["/taotale"]
