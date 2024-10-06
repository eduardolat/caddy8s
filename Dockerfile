# To make sure we have the golang binaries
FROM golang:1.23.1-bookworm AS golang

# Set the base image and basic environment variables
FROM debian:12.7
ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="$PATH:/usr/local/go/bin"
WORKDIR /app/temp

# Copy the golang binaries
COPY --from=golang /usr/local/go /usr/local/go

# Install system dependencies
RUN apt-get update && \
    apt-get install -y wget tzdata git && \
    rm -rf /var/lib/apt/lists/*

# Install downloadable binaries
RUN \
    # Install task
    wget --no-verbose https://github.com/go-task/task/releases/download/v3.38.0/task_linux_amd64.tar.gz && \
    tar -xzf task_linux_amd64.tar.gz && \
    mv ./task /usr/local/bin/task && \
    chmod +x /usr/local/bin/task && \
    # Install Cloudflared
    wget --no-verbose https://github.com/cloudflare/cloudflared/releases/download/2024.9.1/cloudflared-linux-amd64 && \
    mv ./cloudflared-linux-amd64 /usr/local/bin/cloudflared && \
    chmod +x /usr/local/bin/cloudflared && \
    # Install Caddy server
    wget --no-verbose https://github.com/caddyserver/caddy/releases/download/v2.8.4/caddy_2.8.4_linux_amd64.tar.gz && \
    tar -xzf caddy_2.8.4_linux_amd64.tar.gz && \
    mv ./caddy /usr/local/bin/caddy && \
    chmod +x /usr/local/bin/caddy

# Delete the temporary directory and go to the app directory
RUN rm -rf /app/temp
WORKDIR /app

# Set the environment variables
ENV CLOUDFLARED_TOKEN=""
ENV CADDY_CONFIG=""

##############
# START HERE #
##############

copy . /app

EXPOSE 80
CMD ["go", "run", "main.go"]
