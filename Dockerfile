ARG BIN_NAME=feedfilter
ARG BIN_VERSION=
FROM golang:1-alpine AS builder
ARG BIN_NAME
ARG BIN_VERSION
RUN update-ca-certificates
WORKDIR /src/${BIN_NAME}
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-X main.version=${BIN_VERSION}" -o ./out/${BIN_NAME} .

FROM scratch
ARG BIN_NAME
ARG BIN_VERSION
COPY --from=builder /src/${BIN_NAME}/out/${BIN_NAME} /usr/bin/${BIN_NAME}
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/usr/bin/feedfilter"]
LABEL license="GPL-3.0"
LABEL maintainer="Chris Dzombak <chris@dzombak.com>"
LABEL org.opencontainers.image.authors="Chris Dzombak <chris@dzombak.com>"
LABEL org.opencontainers.image.url="https://github.com/cdzombak/${BIN_NAME}"
LABEL org.opencontainers.image.documentation="https://github.com/cdzombak/${BIN_NAME}/blob/main/README.md"
LABEL org.opencontainers.image.source="https://github.com/cdzombak/${BIN_NAME}.git"
LABEL org.opencontainers.image.version="${BIN_VERSION}"
LABEL org.opencontainers.image.licenses="GPL-3.0"
LABEL org.opencontainers.image.title="${BIN_NAME}"
LABEL org.opencontainers.image.description="A versatile RSS feed filtering and manipulation tool."
