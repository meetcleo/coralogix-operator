# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

# Define image version
ARG VERSION=1.0.0
ARG RELEASE=1

# Define image labels
LABEL name="coralogix-operator" \
      vendor="Coralogix Ltd." \
      version="${VERSION}" \
      release="${RELEASE}" \
      summary="Coralogix Operator for special FluentD deployment" \
      description="Coralogix Operator for special FluentD deployment"

# Setup main user envs
ENV USER_UID=1001 \
    USER_NAME=coralogix-operator

# Create main user
RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd && \
    mkdir -p "${HOME}" && \
    chown "${USER_UID}:0" "${HOME}" && \
    chmod ug+rwx "${HOME}"

# Set user
USER ${USER_UID}

# Setup main process
WORKDIR /
COPY LICENSE /licenses/
COPY --from=builder /workspace/manager .

# Setup entrypoint
ENTRYPOINT ["/manager"]
