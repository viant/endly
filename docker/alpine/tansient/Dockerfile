FROM golang:1.12-alpine3.9

ENV NOTVISIBLE "in users profile"
ENV GO111MODULE=on

RUN apk add --no-cache curl python jq make gcc musl-dev git docker bash openssh-server


RUN echo 'root:dev' | chpasswd && \
    sed -i s/#PermitRootLogin.*/PermitRootLogin\ yes/ /etc/ssh/sshd_config && \
    ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
    ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa
RUN apk --no-cache add ca-certificates wget


WORKDIR /
RUN git clone https://github.com/viant/endly.git
WORKDIR /endly/endly

RUN cd /endly/endly && go build endly.go
RUN cp endly /bin/


RUN [ "/bin/bash", "-c", "mkdir -p /var/run/sshd" ]
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]