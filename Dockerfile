FROM golang:1.10 as builder

# Get dep
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/github.com/exavolt/xmpp-server

# Get the dependencies so it can be cached into a layer
COPY Gopkg.lock Gopkg.toml ./
RUN dep ensure -v -vendor-only

ARG revisionID=unknown
ARG buildTimestamp=unknown

# Now copy all the source...
COPY . .

# ...and build it.
RUN CGO_ENABLED=0 go build -o ./bin/xmpp-server \
    -ldflags="-X main.revisionID=${revisionID} -X main.buildTimestamp=${buildTimestamp}" \
    ./cmd/xmpp-server

# Build the runtime image
FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/exavolt/xmpp-server/bin/xmpp-server ./service

# XMPP
EXPOSE 5222

ENTRYPOINT ["./service"]
