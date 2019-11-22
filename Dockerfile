############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git pkgconfig bash build-base
RUN git clone https://github.com/edenhill/librdkafka.git && \
	cd librdkafka && \
	./configure --prefix /usr && \
	make && \
	make install
WORKDIR $GOPATH/src/producer

COPY  producer.go .

# Fetch dependencies.
# Using go get.
RUN go get "github.com/sirupsen/logrus" "github.com/stretchr/testify/require" "github.com/confluentinc/confluent-kafka-go/kafka" "gopkg.in/alecthomas/kingpin.v2"
RUN go get -d -v

# Build the binary.
RUN go build -o /go/bin/ -tags static all 

############################
# STEP 2 build a small image
############################
FROM alpine:3.10

# Copy our static executable.
COPY --from=builder /go/bin/producer /go/bin/producer

# Run the hello binary.
ENTRYPOINT ["/go/bin/producer"]

