FROM openanolis/anolisos:8.4-x86_64

RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /home/admin

ENV TINI_VERSION v0.19.0
ADD ./tini /tini
RUN chmod +x /tini


RUN useradd -m admin
ENV OBAGENT_VERSION 1.1.1
ENV OBAGENT_RELEASE 2

RUN rpm -ivh https://mirrors.aliyun.com/oceanbase/community/stable/el/8/x86_64/obagent-${OBAGENT_VERSION}-${OBAGENT_RELEASE}.el8.x86_64.rpm
RUN chown -R admin:admin /home/admin/obagent
ADD ./replace_yaml.sh /home/admin/obagent

ENTRYPOINT ["/tini", "--"]
CMD ["bash", "-c", " cd /home/admin/obagent && ./replace_yaml.sh  && ./bin/monagent -c conf/monagent.yaml"]