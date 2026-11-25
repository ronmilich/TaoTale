FROM golang:1.27.4-bookworm AS sdk
RUN apt-get update && apt-get install -y --no-install-recommends libasound2 ffmpeg && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL https://csspeechstorage.blob.core.windows.net/drop/1.53.0/SpeechSDK-Linux-1.53.0.tar.gz -o /tmp/speech.tgz \
    && mkdir /opt/speech && tar xzf /tmp/speech.tgz --strip-components=1 -C /opt/speech && rm /tmp/speech.tgz
ENV CGO_ENABLED=1 CGO_CFLAGS=-I/opt/speech/include/c_api CGO_LDFLAGS="-L/opt/speech/lib/x64 -lMicrosoft.CognitiveServices.Speech.core" LD_LIBRARY_PATH=/opt/speech/lib/x64
WORKDIR /spike
COPY go.mod ./
RUN go mod download

FROM sdk
COPY *.go go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go test ./... && go build -o /usr/local/bin/probe .
ENTRYPOINT ["probe"]
