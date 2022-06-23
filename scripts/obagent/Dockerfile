FROM golang:1.16 as builder

WORKDIR /workspace

RUN echo '----------- DATE_CHANGE: Add this line to disable cache during docker building -------------------'

RUN mkdir -p /home/admin/obagent/conf
# Copy the Go Modules manifests
RUN git clone https://github.com/oceanbase/obagent.git

# Build
RUN cd /workspace/obagent && make build-release

FROM openanolis/anolisos:8.4-x86_64

RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /home/admin

ENV TINI_VERSION v0.19.0
ADD ./tini /tini
RUN chmod +x /tini

RUN useradd -m admin

RUN mkdir -p /home/admin/obagent/bin
RUN mkdir -p /home/admin/obagent/run
RUN mkdir -p /home/admin/obagent/log

COPY --from=builder /workspace/obagent/bin/monagent ./obagent/bin
COPY --from=builder /workspace/obagent/etc ./obagent/conf

RUN chown -R admin:admin /home/admin/obagent
ADD ./replace_yaml.sh /home/admin/obagent

ENTRYPOINT ["/tini", "--"]
CMD ["bash", "-c", " cd /home/admin/obagent && ./replace_yaml.sh  && ./bin/monagent -c conf/monagent.yaml"]
