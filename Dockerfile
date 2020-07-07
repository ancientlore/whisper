FROM golang:1.14.4 as builder
WORKDIR /go/src/github.com/ancientlore/whisper
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go get .
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go install
WORKDIR /go/foo
RUN echo "root:x:0:0:user:/home:/bin/bash" > passwd && echo "nobody:x:65534:65534:user:/home:/bin/bash" >> passwd
RUN echo "root:x:0:" > group && echo "nobody:x:65534:" >> group

FROM gcr.io/distroless/static:latest
COPY --from=builder /go/foo/group /etc/group
COPY --from=builder /go/foo/passwd /etc/passwd
COPY --from=builder /go/bin/whisper /usr/local/bin/whisper
EXPOSE 8000
USER nobody:nobody
ENV ROOT /home/www
ENTRYPOINT ["/usr/local/bin/whisper"]
