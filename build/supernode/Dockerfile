# We use this stage for minimizing layers
FROM busybox:latest as SRC
# Copy configs and the source code
COPY ./build/supernode/docker /tmp/docker
COPY ./src/supernode /tmp/supernode-sourcecode

FROM centos:centos7

# Set environments
ENV JAVA_HOME=/usr/java/default
ENV MAVEN_HOME=/usr/local/maven
ENV CATALINA_HOME=/usr/local/tomcat
ENV PATH=$JAVA_HOME/bin:$MAVEN_HOME/bin:$CATALINA_HOME/bin:$PATH

# That's all we need
COPY --from=SRC /tmp /tmp

# Let's go
RUN export JDK_URL=http://download.oracle.com/otn-pub/java/jdk/8u171-b11/512cd62ec5174c3487ac17c61aaa89e8/jdk-8u171-linux-x64.rpm && \
    export MAVEN3_URL=http://www.eu.apache.org/dist/maven/maven-3/3.3.9/binaries/apache-maven-3.3.9-bin.tar.gz && \
    export TOMCAT_URL=https://archive.apache.org/dist/tomcat/tomcat-7/v7.0.84/bin/apache-tomcat-7.0.84.tar.gz && \
    # Download jdk, maven and tomcat
    cp /tmp/docker/repos/* /etc/yum.repos.d/ && \
    yum install -y nginx tar wget && \
    wget --progress=bar:force \
    --header "Cookie: oraclelicense=accept-securebackup-cookie" \
    ${JDK_URL} -O /tmp/jdk.rpm && \
    wget --progress=bar:force ${MAVEN3_URL} -O /tmp/maven.tgz && \
    wget --progress=bar:force ${TOMCAT_URL} -O /tmp/tomcat.tgz && \
    # Install Jdk8
    rpm -i /tmp/jdk.rpm && \
    # Install Maven3
    mkdir /usr/local/maven && \
    tar xzf /tmp/maven.tgz -C /usr/local/maven --strip-components 1 && \
    # Install Tomcat7
    mkdir /usr/local/tomcat && \
    tar xzf /tmp/tomcat.tgz -C /usr/local/tomcat --strip-components 1 && \
    # Build Source Code
    cd /tmp/supernode-sourcecode && \
    mvn clean -U install -DskipTests=true -q && \
    rm -rf $CATALINA_HOME/webapps/* && \
    mv /tmp/supernode-sourcecode/target/supernode.war $CATALINA_HOME/webapps/supernode.war && \
    # Copy nginx and tomcat config
    cp /tmp/docker/nginx/nginx.conf /etc/nginx/nginx.conf && \
    cp /tmp/docker/tomcat/server.xml /usr/local/tomcat/conf/server.xml && \
    # Clean
    rm -rf /tmp/* && \
    rm -rf ~/.m2/repository && \
    yum clean all && \
    rm -rf /var/cache/yum

EXPOSE 8001 8002

CMD nginx && catalina.sh run
