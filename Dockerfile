# syntax=docker/dockerfile:1

FROM golang:1.24 as build

# Set destination for COPY
WORKDIR /src

COPY . .
RUN go mod download
RUN go build

FROM alpine as run
# Download Go modules
# Build
COPY --from=build /src/CraftyProxy /app/CraftyProxy
WORKDIR /app
RUN chmod +x ./CraftyProxy

# expose enough ports for 10 servers
EXPOSE 25565-25575

# Run
CMD ["/CraftyProxy"]

LABEL org.opencontainers.image.authors="button@bttn.dev"
LABEL org.opencontainers.image.source="https://github.com/Button24/CraftyProxy"
