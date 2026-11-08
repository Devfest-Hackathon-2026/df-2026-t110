# syntax=docker/dockerfile:1

ARG GO_VERSION=1.27.3
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src/Server
ENV GOTOOLCHAIN=local

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=Server/go.mod,target=go.mod \
    --mount=type=bind,source=Server/go.sum,target=go.sum \
    go mod download

COPY Server/ /src/Server/
COPY Textbooks/pages/ /src/Server/cmd/textbooks/

ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -tags textbook_embed -o /bin/server ./cmd

FROM alpine:latest AS final
RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add ca-certificates tzdata && update-ca-certificates

ARG UID=10001
RUN adduser --disabled-password --gecos "" --home "/nonexistent" \
    --shell "/sbin/nologin" --no-create-home --uid "${UID}" appuser
USER appuser

COPY --from=build /bin/server /bin/
ENTRYPOINT ["/bin/server"]
