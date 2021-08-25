
FROM golang:alpine AS builder

RUN apk update
RUN apk add git libc-dev gcc

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download


COPY . .

RUN go mod tidy
RUN go build -o main ./src

WORKDIR /dist
RUN cp /build/main .
WORKDIR /data
ADD revo-bench.yaml .
########################################################
FROM alpine:latest

COPY --chown=0:0 --from=builder /dist /
COPY --chown=65534:0 --from=builder /data /data
USER 65534
WORKDIR /data

EXPOSE 8080:8080


ENTRYPOINT ["/main", "-conf", "revo-bench.yaml"]