ARG GO_VERSION=1.24
ARG IMG_VERSION=1.24

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS builder
WORKDIR /go/src/github.com/ancientlore/whisper
COPY . .
RUN go version
ARG TARGETOS TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -o /go/bin/whisper

FROM ancientlore/goimg:${IMG_VERSION}
COPY --from=builder /go/bin/whisper /usr/local/bin/whisper
COPY example/static/dog.png /home/www/whisper.png
COPY example/.index-docker.md /home/www/index.md
EXPOSE 8080
ENV ROOT=/home/www
ENTRYPOINT ["/usr/local/bin/whisper"]
