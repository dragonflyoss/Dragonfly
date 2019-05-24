FROM golang:1.10.4-alpine as builder

WORKDIR /go/src/github.com/dragonflyoss/Dragonfly
RUN apk --no-cache add bash make gcc libc-dev git

COPY . /go/src/github.com/dragonflyoss/Dragonfly

# go build dfdaemon and dfget.
# write the resulting executable to the dir /dfclient.
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o /dfclient/dfdaemon cmd/dfdaemon/main.go
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o /dfclient/dfget cmd/dfget/main.go

FROM alpine:3.8

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /dfclient /dfclient

# dfdaemon will listen 65001 in default.
EXPOSE 65001

# use the https://index.docker.io as default registry.
CMD [ "--registry", "https://index.docker.io" ]

ENTRYPOINT [ "/dfclient/dfdaemon" ]
