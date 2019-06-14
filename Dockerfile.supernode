FROM golang:1.10.4-alpine as builder

WORKDIR /go/src/github.com/dragonflyoss/Dragonfly
RUN apk --no-cache add bash make gcc libc-dev git

COPY . /go/src/github.com/dragonflyoss/Dragonfly

# go build supernode.
# write the resulting executable to the dir /opt/dragonfly/df-supernode.
RUN make build-supernode && make install-supernode

FROM nginx:1.16-alpine

RUN apk --no-cache add ca-certificates bash

COPY --from=builder /go/src/github.com/dragonflyoss/Dragonfly/hack/supernode-nginx.conf /etc/nginx/nginx.conf
COPY --from=builder /opt/dragonfly/df-supernode/supernode /opt/dragonfly/df-supernode/supernode

# supernode will listen 8001,8002 in default.
EXPOSE 8001 8002

ENTRYPOINT [ "sh", "-c", "nginx && /opt/dragonfly/df-supernode/supernode" ]
