FROM golang:1.10.4-alpine as builder

WORKDIR /go/src/github.com/dragonflyoss/Dragonfly
RUN apk --no-cache add bash make gcc libc-dev git

COPY . /go/src/github.com/dragonflyoss/Dragonfly

# make build dfdaemon and dfget.
# write the resulting executable to the dir /opt/dragonfly/df-client.
RUN make build-client && make install-client

FROM alpine:3.8

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /opt/dragonfly/df-client /opt/dragonfly/df-client

# dfdaemon will listen 65001 in default.
EXPOSE 65001

# use the https://index.docker.io as default registry.
CMD [ "--registry", "https://index.docker.io" ]

ENTRYPOINT [ "/opt/dragonfly/df-client/dfdaemon" ]
