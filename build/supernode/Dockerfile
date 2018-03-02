FROM centos:centos7

COPY ./build/supernode/docker/repos/aliyun-centos-7.repo /etc/yum.repos.d/CentOS-Base.repo
COPY ./build/supernode/docker/repos/aliyun-epel-7.repo /etc/yum.repos.d/epel.repo
COPY ./build/supernode/docker/repos/nginx.repo /etc/yum.repos.d/nginx.repo

# Install Nginx Wget Tar
RUN yum install -y nginx wget tar

# Install Jdk8
RUN wget --no-cookies --no-check-certificate --header "Cookie: oraclelicense=accept-securebackup-cookie" http://download.oracle.com/otn-pub/java/jdk/8u161-b12/2f38c3b165be4555a1fa6e98c45e0808/jdk-8u161-linux-x64.rpm -O /tmp/jdk-8u161-linux-x64.rpm && \
rpm -i /tmp/jdk-8u161-linux-x64.rpm \
    && rm /tmp/jdk-8u161-linux-x64.rpm

# Install Maven3
RUN wget http://www.eu.apache.org/dist/maven/maven-3/3.3.9/binaries/apache-maven-3.3.9-bin.tar.gz -O /tmp/apache-maven-3.3.9-bin.tar.gz \
    && cd /usr/local && tar xzf /tmp/apache-maven-3.3.9-bin.tar.gz \
    && ln -s /usr/local/apache-maven-3.3.9 /usr/local/maven \
    && rm /tmp/apache-maven-3.3.9-bin.tar.gz

# Install Tomcat7
RUN wget https://archive.apache.org/dist/tomcat/tomcat-7/v7.0.84/bin/apache-tomcat-7.0.84.tar.gz -O /tmp/apache-tomcat-7.0.84.tar.gz \
    && cd /usr/local && tar xzf /tmp/apache-tomcat-7.0.84.tar.gz \
    && ln -s /usr/local/apache-tomcat-7.0.84 /usr/local/tomcat \
    && rm /tmp/apache-tomcat-7.0.84.tar.gz

# Set Environment Variables
RUN mv $(find /usr/java/ -maxdepth 1 -name "jdk*" -type d) /usr/java/jdk
ENV JAVA_HOME /usr/java/jdk
ENV MAVEN_HOME /usr/local/maven
ENV CATALINA_HOME /usr/local/tomcat
ENV PATH $JAVA_HOME/bin:$MAVEN_HOME/bin:$CATALINA_HOME/bin:$PATH

# Build Source Code
COPY ./src/supernode /tmp/supernode-sourcecode
RUN cd /tmp/supernode-sourcecode \
    && mvn clean -U install -DskipTests=true \
    && rm -rf $CATALINA_HOME/webapps/* \
    && mv /tmp/supernode-sourcecode/target/supernode.war $CATALINA_HOME/webapps/supernode.war \
    && rm -rf /tmp/supernode-sourcecode \
    && rm -rf ~/.m2/repository

COPY ./build/supernode/docker/tomcat/server.xml /usr/local/tomcat/conf/server.xml
COPY ./build/supernode/docker/nginx/nginx.conf /etc/nginx/nginx.conf
EXPOSE 8001 8002
CMD nginx && catalina.sh run
