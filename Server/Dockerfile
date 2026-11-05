# syntax=docker/dockerfile:1

ARG GO_VERSION=1.27.3
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src
ENV GOTOOLCHAIN=local

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /bin/server ./cmd

FROM alpine:latest AS final
RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add ca-certificates tzdata && update-ca-certificates

ARG UID=10001
RUN adduser --disabled-password --gecos "" --home "/nonexistent" \
    --shell "/sbin/nologin" --no-create-home --uid "${UID}" appuser
USER appuser

COPY --from=build /bin/server /bin/
ENTRYPOINT ["/bin/server"]
