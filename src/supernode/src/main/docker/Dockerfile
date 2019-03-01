# We use this stage for minimizing layers
FROM busybox:latest as SRC
# Copy sources
COPY sources /tmp/sources

FROM centos:7

# That's all we need
COPY --from=SRC /tmp /tmp

RUN yum install -y epel-release && \
    yum install -y -q telnet nginx java-1.8.0-openjdk.x86_64 && \
    cp /tmp/sources/start.sh /root/start.sh && \
    cp /tmp/sources/nginx.conf /etc/nginx/nginx.conf && \
    mkdir -p /home/admin/supernode/bin && \
    cp /tmp/sources/start.sh /home/admin/supernode/bin/ && \
    yum clean all && \
    rm -rf /var/cache/yum && \
    rm -rf /tmp/* && \
    history -c

ADD supernode.jar supernode.jar

EXPOSE 8001 8002

CMD ["-Dsupernode.advertiseIp="]
ENTRYPOINT /root/start.sh $0 $@

