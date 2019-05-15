FROM ubuntu:16.04

RUN apt-get update -y && apt-get install -y git libaio1 libc-bin unixodbc unixodbc-dev pkg-config tzdata unixodbc-dev zip gcc curl bash bash-completion openssh-server

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
RUN rm /etc/localtime && ln -fs /usr/share/zoneinfo/UTC /etc/localtime
WORKDIR /
COPY compact.tar.gz /
RUN tar xvzf compact.tar.gz && rm compact.tar.gz

RUN [ "/bin/bash", "-c", "mkdir -p /var/run/sshd" ]
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]