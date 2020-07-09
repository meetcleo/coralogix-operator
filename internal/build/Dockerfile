FROM registry.access.redhat.com/ubi8/ubi:latest

ARG VERSION=latest

LABEL Description="Special FluentD image for Coralogix integration" \
      Vendor="Coralogix Inc." \
      Version="${VERSION}"

RUN yum install -y hostname sudo make gcc gcc-c++ ruby-devel && \
    curl -L https://toolbelt.treasuredata.com/sh/install-redhat-td-agent3.sh | sh && \
    td-agent-gem install --no-rdoc --no-ri --no-document \
        fluent-plugin-kubernetes_metadata_filter \
        fluent-plugin-multi-format-parser \
        fluent-plugin-concat \
        fluent-plugin-detect-exceptions \
        fluent-plugin-coralogix && \
    td-agent-gem sources --clear-all && \
    rm -rf /tmp/* /var/tmp/* /var/cache/* /opt/td-agent/embedded/lib/ruby/gems/*/cache/*.gem && \
    ulimit -n 65536 && \
    yum autoremove -y make gcc gcc-c++ ruby-devel && \
    yum clean all

ENV LD_PRELOAD=/opt/td-agent/embedded/lib/libjemalloc.so

USER td-agent

COPY configs/td-agent.conf /etc/td-agent/td-agent.conf

CMD ["td-agent"]