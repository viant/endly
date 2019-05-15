FROM ubuntu:16.04

RUN apt-get update -y && apt-get install -y wget libc-bin  build-essential gcc git g++ unixodbc pkg-config unixodbc-dev libaio1 openssh-server curl python-pip bash lsb-release bash-completion pkg-config

ENV NOTVISIBLE "in users profile"
RUN rm /etc/ssh/ssh_host_rsa_key
RUN rm /etc/ssh/ssh_host_dsa_key
RUN ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
    ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa
RUN echo "root:dev" | chpasswd
RUN sed -i 's/PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config
# SSH login fix. Otherwise user is kicked off after login
RUN sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd
RUN echo "export VISIBLE=now" >> /etc/profile
RUN apt-get install -y ca-certificates
RUN pip install --no-cache-dir docker-compose

WORKDIR /


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





WORKDIR /usr/local
RUN wget https://download.docker.com/linux/static/stable/x86_64/docker-18.09.1.tgz && \
    tar xvzf docker-18.09.1.tgz && \
    rm docker-18.09.1.tgz && \
    cp /usr/local/docker/docker /usr/local/bin/docker

RUN wget https://dl.google.com/go/go1.12.1.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.12.1.linux-amd64.tar.gz && \
    rm go1.12.1.linux-amd64.tar.gz && \
    ln -s /usr/local/go/bin/go /usr/local/bin/go


WORKDIR /
RUN git clone https://github.com/viant/endly.git
COPY dep.tar.gz .
RUN tar xvzf dep.tar.gz && \
    rm dep.tar.gz && \
    cp oci8.pc /endly/endly && \
    cp oci8.pc /usr/lib/oracle/12.2/client64/lib/


WORKDIR /endly/endly
ENV LD_LIBRARY_PATH=/usr/lib/oracle/12.2/client64/lib/
ENV PKG_CONFIG_PATH=/endly/endly

RUN sed -i 's/\/\/cgo/ /g' /endly/bootstrap/bootstrap.go
RUN go build endly.go
RUN cp endly /usr/local/bin/

WORKDIR /
RUN tar cvzf /compact.tar.gz /usr/local/bin/endly  /usr/local/bin/docker /usr/local/bin/docker-compose \
    /usr/bin/python2.7 /usr/bin/python2 /usr/bin/python /usr/lib/python2.7 /usr/local/lib/python2.7/dist-packages \
    /usr/local/lib/l* /etc/vertica.ini /etc/odbcinst.ini /opt/vertica /usr/lib/oracle /usr/include/oracle /etc/environment



RUN [ "/bin/bash", "-c", "mkdir -p /var/run/sshd" ]
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]