# Build container
FROM golang:bullseye AS build

# Setup environment
RUN mkdir -p /data
WORKDIR /data

# Build the release
COPY . .
RUN make

# Extract the release
RUN mkdir -p /out
RUN cp out/htorrent /out/htorrent

# Release container
FROM debian:bullseye

# Add certificates
RUN apt update
RUN apt install -y ca-certificates

# Add the release
COPY --from=build /out/htorrent /usr/local/bin/htorrent

CMD /usr/local/bin/htorrent
