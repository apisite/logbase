
ARG GOLANG_IMAGE=ghcr.io/dopos/golang-alpine
ARG GOLANG_VERSION=v1.23.6-alpine3.21.3
ARG APP=logbase

FROM --platform=$BUILDPLATFORM ${GOLANG_IMAGE}:${GOLANG_VERSION} AS build

WORKDIR /opt/app
RUN apk --update add curl git make

# Cached layer
COPY ./go.mod ./go.sum ./
RUN go mod download

# Sources dependent layer
COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=`git describe --tags --always`" -a ./cmd/logbase/
#make build-standalone

#FROM alpine:3.11.2
FROM ghcr.io/dopos/docker-alpine:v3.21.3

ENV DOCKERFILE_VERSION  200110

WORKDIR /opt/app

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /opt/app/logbase /usr/bin/logbase

EXPOSE 8080
ENTRYPOINT ["/usr/bin/logbase"]
