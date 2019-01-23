FROM alpine:3.8

ENV NOTVISIBLE "in users profile"
ENV GO111MODULE=on

RUN apk add --no-cache curl bash make gcc musl-dev git openssh-server
RUN echo 'root:root' | chpasswd && \
    sed -i s/#PermitRootLogin.*/PermitRootLogin\ yes/ /etc/ssh/sshd_config && \
    ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
    ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa

RUN curl -L "https://github.com/docker/compose/releases/download/1.23.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
RUN chmod +x /usr/local/bin/docker-compose


RUN [ "/bin/bash", "-c", "mkdir -p /var/run/sshd" ]
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]