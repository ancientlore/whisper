FROM golang:1.16 as builder
WORKDIR /go/src/github.com/ancientlore/whisper
COPY . .
RUN go version
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go get .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go install

FROM ancientlore/goimg:latest
COPY --from=builder /go/bin/whisper /usr/local/bin/whisper
EXPOSE 8080
ENV ROOT /home/www
ENTRYPOINT ["/usr/local/bin/whisper"]
