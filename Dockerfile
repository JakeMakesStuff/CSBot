FROM golang:1.15-alpine
WORKDIR /var/app
COPY . .
RUN apk add gcc g++
RUN go build -o app

FROM oraclelinux:8
WORKDIR /var/app
RUN yum install -y wget unzip libaio && \
    rm -rf /var/cache/yum
RUN wget https://download.oracle.com/otn_software/linux/instantclient/instantclient-basiclite-linuxx64.zip && \
    unzip instantclient-basiclite-linuxx64.zip && \
    rm -f instantclient-basiclite-linuxx64.zip && \
    cd instantclient* && \
    rm -f *jdbc* *occi* *mysql* *jar uidrvci genezi adrci && \
    echo /opt/oracle/instantclient* > /etc/ld.so.conf.d/oracle-instantclient.conf && \
    ldconfig && cd /var/app
COPY --from=0 /var/app .
ENTRYPOINT ./app
