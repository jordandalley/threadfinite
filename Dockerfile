# First stage. Building the threadfin binary
# -----------------------------------------------------------------------------
FROM golang:bookworm AS builder-go

ARG BUILD_DATE
ARG VCS_REF
ARG THREADFIN_PORT=34400
ARG THREADFIN_VERSION

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY threadfin.go ./
COPY src src

# Rebuild the html files into src/webUI.go file prior to recompile
COPY html html
COPY build-html.go ./
RUN go mod vendor && go run build-html.go

# Build the application with optimizations
RUN CGO_ENABLED=0 go build -mod=mod -ldflags="-s -w" -trimpath -o threadfin threadfin.go

# Second state. Building the wrapper binary
# -----------------------------------------------------------------------------
FROM python:bookworm AS builder-py

ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY src/internal/wrapper/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY src/internal/wrapper/wrapper.py .

RUN pyinstaller --name wrapper --onefile wrapper.py

# Last stage. Creating a minimal image
# -----------------------------------------------------------------------------

FROM debian:bookworm-slim AS standard

LABEL org.label-schema.build-date="${BUILD_DATE}" \
      org.label-schema.name="Threadfin" \
      org.label-schema.description="Dockerised Threadfin" \
      org.label-schema.url="" \
      org.label-schema.vcs-ref="${VCS_REF}" \
      org.label-schema.vcs-url="https://github.com/jordandalley/Threadfin" \
      org.label-schema.vendor="Threadfin" \
      org.label-schema.version="${THREADFIN_VERSION}" \
      org.label-schema.schema-version="1.0" \
      DISCORD_URL=""

ENV THREADFIN_BIN=/home/threadfin/bin \
    THREADFIN_CONF=/home/threadfin/conf \
    THREADFIN_HOME=/home/threadfin \
    THREADFIN_TEMP=/tmp/threadfin \
    THREADFIN_CACHE=/home/threadfin/cache \
    THREADFIN_UID=31337 \
    THREADFIN_GID=31337 \
    THREADFIN_USER=threadfin \
    THREADFIN_BRANCH=main \
    THREADFIN_DEBUG=0 \
    THREADFIN_PORT=34400 \
    THREADFIN_LOG=/var/log/threadfin.log \
    THREADFIN_BIND_IP_ADDRESS=0.0.0.0 \
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/home/threadfin/bin \
    DEBIAN_FRONTEND=noninteractive

# Set working directory
WORKDIR $THREADFIN_HOME

# Arguments to add the jellyfin repository
ARG TARGETARCH
ARG OS_VERSION=debian
ARG OS_CODENAME=bookworm

# Copy threadfin binary out of builder-go container into standard
COPY --from=builder-go /app/threadfin $THREADFIN_BIN/
# Copy wrapper binary out of builder-py container into standard
COPY --from=builder-py /app/dist/wrapper $THREADFIN_BIN/

# Install base level packages, then install jellyfin repo before installing jellyfin-ffmpeg, cleaning up and settings binaries as executable
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl tzdata gnupg apt-transport-https && \
    curl -fsSL https://repo.jellyfin.org/jellyfin_team.gpg.key | gpg --dearmor -o /etc/apt/trusted.gpg.d/debian-jellyfin.gpg && \
    echo "deb [arch=${TARGETARCH}] https://repo.jellyfin.org/master/${OS_VERSION} ${OS_CODENAME} main" > /etc/apt/sources.list.d/jellyfin.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends --no-install-suggests jellyfin-ffmpeg7 && \
    apt-get remove gnupg apt-transport-https --yes && \
    apt-get clean autoclean --yes && \
    apt-get autoremove --yes && \
    rm -rf /var/cache/apt/archives* /var/lib/apt/lists/* && \
    mkdir -p $THREADFIN_BIN $THREADFIN_CONF $THREADFIN_TEMP $THREADFIN_HOME/cache && \
    chmod a+rwX $THREADFIN_CONF $THREADFIN_TEMP && \
    chmod +rx $THREADFIN_BIN/threadfin && \
    chmod +rx $THREADFIN_BIN/wrapper

# Configure container volume mappings
VOLUME $THREADFIN_CONF
VOLUME $THREADFIN_TEMP
EXPOSE $THREADFIN_PORT

# start threadfin
ENTRYPOINT ["sh", "-c", "${THREADFIN_BIN}/threadfin -port=${THREADFIN_PORT} -bind=${THREADFIN_BIND_IP_ADDRESS} -config=${THREADFIN_CONF} -debug=${THREADFIN_DEBUG}"]
