# syntax=docker/dockerfile:1

FROM golang:1.24

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY crafty/ crafty/
COPY proxy/ proxy/
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /crafty-proxy

# expose enough ports for 10 servers
EXPOSE 25565-25575

# Run
CMD ["/crafty-proxy"]

LABEL org.opencontainers.image.authors="button@bttn.dev"
LABEL org.opencontainers.image.source="https://github.com/Button24/CraftyProxy"
