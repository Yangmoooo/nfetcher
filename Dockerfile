FROM golang:1.25 AS build
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ENV HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY} \
    GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/nfetcher ./cmd/nfetcher

FROM debian:stable-slim
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown
ARG APT_DEBIAN_MIRROR=http://mirrors.tuna.tsinghua.edu.cn/debian
ARG APT_SECURITY_MIRROR=http://mirrors.tuna.tsinghua.edu.cn/debian-security
ENV HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY}
RUN if [ -n "${APT_DEBIAN_MIRROR}" ]; then \
        sed -i \
        -e "s|http://deb.debian.org/debian|${APT_DEBIAN_MIRROR}|g" \
        -e "s|https://deb.debian.org/debian|${APT_DEBIAN_MIRROR}|g" \
        /etc/apt/sources.list.d/debian.sources; \
    fi \
    && if [ -n "${APT_SECURITY_MIRROR}" ]; then \
        sed -i \
        -e "s|http://deb.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        -e "s|https://deb.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        /etc/apt/sources.list.d/debian.sources; \
    fi \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/nfetcher /usr/local/bin/nfetcher

LABEL org.opencontainers.image.title="nfetcher" \
      org.opencontainers.image.source="https://github.com/Yangmoooo/nfetcher" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}"

ENV TZ=Asia/Shanghai
ENTRYPOINT ["nfetcher"]
CMD ["daemon"]
