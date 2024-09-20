# syntax=docker/dockerfile:1

# We use the latest Go 1.x version unless asked to use something else.
# The GitHub Actions CI job sets this argument for a consistent Go version.
ARG GO_VERSION=1

# Setup the base environment. The BUILDPLATFORM is set automatically by Docker.
# The --platform=${BUILDPLATFORM} flag tells Docker to build the function using
# the OS and architecture of the host running the build, not the OS and
# architecture that we're building the function for.
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS build

RUN apt-get update && apt-get install -y coreutils jq unzip zsh less
RUN mkdir /scripts /.aws && chown 2000:2000 /scripts /.aws

RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "/tmp/awscliv2.zip" && \
	unzip "/tmp/awscliv2.zip" && \
	./aws/install

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

# Produce the Function image. We use a very lightweight 'distroless'
# Python3 image that includes useful commands but not build tools used
# in previous stages.
# FROM python:3.12
FROM gcr.io/distroless/python3-debian12 AS image

WORKDIR /
COPY --from=build --chown=2000:2000 /scripts /scripts
COPY --from=build --chown=2000:2000 /.aws /.aws

COPY --from=build /bin /bin
COPY --from=build /etc /etc
COPY --from=build /lib /lib
COPY --from=build /tmp /tmp
COPY --from=build /usr /usr
COPY --from=build /function /function
EXPOSE 9443
USER nonroot:nonroot
ENTRYPOINT ["/function"]
