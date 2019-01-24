FROM alpine:3.8

ENV NOTVISIBLE "in users profile"
ENV GO111MODULE=on


RUN apk add --no-cache curl bash make gcc musl-dev ncurses py-pip  git openssh-server
RUN echo 'root:dev' | chpasswd && \
    sed -i s/#PermitRootLogin.*/PermitRootLogin\ yes/ /etc/ssh/sshd_config && \
    ssh-keygen -f /etc/ssh/ssh_host_rsa_key -N '' -t rsa && \
    ssh-keygen -f /etc/ssh/ssh_host_dsa_key -N '' -t dsa

COPY endly /usr/bin/endly
COPY docker /usr/bin/docker
COPY usr/lib/ /usr/lib/
COPY usr/local/ /usr/local/
COPY lib/ lib/
COPY docker-compose /usr/bin/docker-compose

RUN [ "/bin/bash", "-c", "mkdir -p /var/run/sshd" ]
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]