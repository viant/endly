FROM golang:1.11.4-alpine3.8

ENV NOTVISIBLE "in users profile"
ENV GO111MODULE=on

RUN apk add --no-cache curl py-pip jq make gcc musl-dev git docker wget bash geoip-dev openssh-server sudo && \
    pip install --no-cache-dir docker-compose

RUN echo 'root:dev' | chpasswd && \
    sed -i s/#PermitRootLogin.*/PermitRootLogin\ yes/ /etc/ssh/sshd_config && \
    ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
    ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa

RUN wget http://www.unixodbc.org/unixODBC-2.3.5.tar.gz &&\
    tar xzvf unixODBC-2.3.5.tar.gz &&\
    cd /usr/local/lib/ && \
    ln -s libodbc.so.2.0.0 libodbc.so.1 && \
    ln -s libodbcinst.so.2.0.0 libodbcinst.so.1 && \
    cd - &&\
    cd unixODBC-2.3.5 &&\
    ./configure --sysconfdir=/etc --disable-gui --disable-drivers --enable-iconv --with-iconv-char-enc=UTF8 --with-iconv-ucode-enc=UTF16LE &&\
    make &&\
    make install &&\
    cd .. && \
    rm -rf unixODBC-2.3.5 unixODBC-2.3.5.tar.gz


WORKDIR /
RUN git clone https://github.com/viant/endly.git
WORKDIR /endly/endly

RUN sed -i 's/\/\/cgo/ /g' /endly/bootstrap/bootstrap.go

RUN cd /endly/endly && go build endly.go
RUN cp endly /bin/


RUN [ "/bin/bash", "-c", "mkdir -p /var/run/sshd" ]
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]