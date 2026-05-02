FROM golang:1.22-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS="-trimpath"

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o /out/tools-api ./

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git wget patch \
    && adduser -D -u 10001 forge
WORKDIR /workspace
COPY --from=build /out/tools-api /usr/local/bin/tools-api

ENV WORKSPACE_DIR=/workspace \
    LISTEN_ADDR=:8088 \
    SAFE_MODE=true

EXPOSE 8088
USER forge
ENTRYPOINT ["/usr/local/bin/tools-api"]
