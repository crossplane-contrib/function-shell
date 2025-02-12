# syntax=docker/dockerfile:1

# We use the latest Go 1.x version unless asked to use something else.
# The GitHub Actions CI job sets this argument for a consistent Go version.
ARG GO_VERSION=1.23

# Setup the base environment. The BUILDPLATFORM is set automatically by Docker.
# The --platform=${BUILDPLATFORM} flag tells Docker to build the function using
# the OS and architecture of the host running the build, not the OS and
# architecture that we're building the function for.
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS build

# Download platform-specific AWS CLI binaries
ARG TARGETPLATFORM

WORKDIR /fn

# Most functions don't want or need CGo support, so we disable it.
ENV CGO_ENABLED=0

# We run go mod download in a separate step so that we can cache its results.
# This lets us avoid re-downloading modules if we don't need to. The type=target
# mount tells Docker to mount the current directory read-only in the WORKDIR.
# The type=cache mount tells Docker to cache the Go modules cache across builds.
RUN --mount=target=. --mount=type=cache,target=/go/pkg/mod go mod download

# The TARGETOS and TARGETARCH args are set by docker. We set GOOS and GOARCH to
# these values to ask Go to compile a binary for these architectures. If
# TARGETOS and TARGETOS are different from BUILDPLATFORM, Go will cross compile
# for us (e.g. compile a linux/amd64 binary on a linux/arm64 build machine).
ARG TARGETOS
ARG TARGETARCH

# Build the function binary. The type=target mount tells Docker to mount the
# current directory read-only in the WORKDIR. The type=cache mount tells Docker
# to cache the Go modules cache across builds.
RUN --mount=target=. \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /function .

# Produce the Function image.
FROM alpine:3.21.2 AS image

ENV KUBECTL_VERSION=1.29.11
ENV GH_CLI_VERSION=2.65.0
ENV BOILERPLATE_VERSION=0.5.19
ENV HELM_DOCS_VERSION=1.14.2

RUN apk update && apk add --no-cache \
  ca-certificates \
  bash \
  curl \
  git \
  jq \
  pre-commit \
  && rm -rf /var/cache/apk/*

RUN curl -fsSL "https://dl.k8s.io/release/v$KUBECTL_VERSION/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl \
  && curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_linux_amd64.tar.gz" -o /tmp/gh.tar.gz \
  && tar xzf /tmp/gh.tar.gz \
  && chmod +x gh_${GH_CLI_VERSION}_linux_amd64/bin/gh \
  && mv gh_${GH_CLI_VERSION}_linux_amd64/bin/gh /usr/local/bin/ \
  && rm /tmp/gh.tar.gz \
  && rm -rf ./gh_${GH_CLI_VERSION}_linux_amd64 \
  && curl -fsSL "https://github.com/gruntwork-io/boilerplate/releases/download/v${BOILERPLATE_VERSION}/boilerplate_linux_amd64" -o /usr/local/bin/boilerplate && chmod +x /usr/local/bin/boilerplate \
  && curl -fsSL "https://github.com/norwoodj/helm-docs/releases/download/v${HELM_DOCS_VERSION}/helm-docs_${HELM_DOCS_VERSION}_Linux_x86_64.tar.gz" -o /tmp/helm-docs.tar.gz && tar xzf /tmp/helm-docs.tar.gz && mv helm-docs /usr/local/bin/ && rm /tmp/helm-docs.tar.gz

WORKDIR /
COPY --from=build /function /function
EXPOSE 9443
RUN addgroup -g 65532 nonroot && adduser -u 65532 -G nonroot -h /home/nonroot -S -D -s /usr/sbin/nologin nonroot
USER nonroot:nonroot
ENTRYPOINT ["/function"]
