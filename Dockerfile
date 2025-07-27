# syntax=docker/dockerfile:1

FROM golang:1.24

# Set destination for COPY
WORKDIR /app

# Download Go modules

# Build
RUN CGO_ENABLED=0 GOOS=linux go install github.com/Botond24/CraftyProxy@latest

# expose enough ports for 10 servers
EXPOSE 25565-25575

# Run
CMD ["CraftyProxy"]

LABEL org.opencontainers.image.authors="button@bttn.dev"
LABEL org.opencontainers.image.source="https://github.com/Button24/CraftyProxy"
