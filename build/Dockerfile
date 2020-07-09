FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ARG VERSION=1.0.0
ARG RELEASE=1

LABEL name="coralogix-operator" \
      vendor="Coralogix Inc." \
      version="${VERSION}" \
      release="${RELEASE}" \
      summary="Coralogix Operator for special FluentD deployment" \
      description="Coralogix Operator for special FluentD deployment"

ENV OPERATOR=/usr/local/bin/coralogix-operator \
    USER_UID=1001 \
    USER_NAME=coralogix-operator

COPY LICENSE /licenses/
COPY build/_output/bin/coralogix-operator ${OPERATOR}
COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]
USER ${USER_UID}
