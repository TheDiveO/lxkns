# Please note that the Alpine version also controls the Golang version for the
# golang images.
ARG ALPINE_VERSION=3.12

# 1st stage: build the lxkns binary, now this requires cgo and we thus need gcc
# ... and then we also need header files. Oh, well. Caching to the rescue; we
# start with the gcc and header stuff, which is kind of base builder image stuff
# anyway.
FROM golang:alpine${ALPINE_VERSION} AS builder
WORKDIR /src
RUN apk add --no-cache --update-cache gcc musl-dev
# We now try to cache only the dependencies in a separate layer, so we can speed
# up things in case the dependencies do not change. This then reduces the amount
# of fetching and compiling required when compiling the final binary later.
COPY go.mod go.sum ./
RUN go mod graph | awk '{if ($1 !~ "@") print $2}' | xargs go get
# And now, finally, we build the lxkns service itself.
COPY . .
RUN go build -v -o /lxkns ./cmd/lxkns

# 2nd and final stage: create the final image containing only the lxkns binary
# as well as required libraries.
FROM alpine:${ALPINE_VERSION}
COPY --from=builder /lxkns /
RUN apk add --no-cache \
        libcap && \
    setcap "cap_sys_admin,cap_sys_chroot,cap_sys_ptrace+ep" /lxkns && \
    apk del --no-cache libcap
USER 65534
CMD ["/lxkns"]