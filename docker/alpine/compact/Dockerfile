FROM alpine:3.8

ENV NOTVISIBLE "in users profile"
ENV GO111MODULE=on


RUN apk add --no-cache curl py-pip bash bash-completion make gcc unixodbc ncurses git openssh-server
RUN pip install --no-cache-dir docker-compose
RUN echo 'root:dev' | chpasswd && \
    sed -i s/#PermitRootLogin.*/PermitRootLogin\ yes/ /etc/ssh/sshd_config && \
    ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
    ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa

WORKDIR /
ADD . .
ENV LD_LIBRARY_PATH=/usr/local/lib

RUN [ "/bin/bash", "-c", "mkdir -p /var/run/sshd" ]
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]