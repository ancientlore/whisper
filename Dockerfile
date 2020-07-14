FROM golang:1.14 as builder
WORKDIR /go/src/github.com/ancientlore/whisper
COPY . .
RUN go version
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go get .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go install

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /go/bin/whisper /usr/local/bin/whisper
EXPOSE 8000
ENV ROOT /home/www
ENTRYPOINT ["/usr/local/bin/whisper"]
