FROM golang:1.10

COPY . /go/src/github.com/alibaba/Dragonfly

WORKDIR /go/src/github.com/alibaba/Dragonfly/build

RUN ./build.sh client && cd client && make install

# dfdaemon will listen 65001 in dafault.
EXPOSE 65001

ENTRYPOINT ["dfdaemon"]
