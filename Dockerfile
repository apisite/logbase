
# FROM golang:1.13.5-buster as builder
FROM golang:1.13.4-alpine3.10 as builder

WORKDIR /opt/app
RUN apk --update add curl git make

# Cached layer
COPY ./go.mod ./go.sum ./
RUN go mod download

# Sources dependent layer
COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=`git describe --tags --always`" -a ./cmd/logbase/
#make build-standalone

FROM alpine:3.10

ENV DOCKERFILE_VERSION  200108

WORKDIR /opt/app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /opt/app/logbase /usr/bin/logbase

ENTRYPOINT ["/usr/bin/logbase"]
