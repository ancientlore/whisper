ARG GO_VERSION=1.23
ARG IMG_VERSION=1.23

FROM golang:${GO_VERSION} AS builder
WORKDIR /go/src/github.com/ancientlore/whisper
COPY . .
RUN go version
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go get .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go install

FROM ancientlore/goimg:${IMG_VERSION}
COPY --from=builder /go/bin/whisper /usr/local/bin/whisper
COPY example/static/dog.png /home/www/whisper.png
COPY example/.index-docker.md /home/www/index.md
EXPOSE 8080
ENV ROOT=/home/www
ENTRYPOINT ["/usr/local/bin/whisper"]
