# octopilot Docker Image
# This is a 2-steps build using "multistage builds"
# https://docs.docker.com/engine/userguide/eng-image/multistage-build/

###########################
# 1st step: build the app #
###########################
FROM golang:1.13 AS builder

COPY . /workspace
WORKDIR /workspace
RUN CGO_ENABLED=0 go build -mod=vendor -ldflags "-extldflags -static -linkmode internal"

####################################
# 2nd step: define the "run" image #
####################################
FROM alpine:3.11

LABEL maintainer="https://github.com/dailymotion/octopilot" \
      name="OctoPilot"

RUN echo "Installing root certificates" \
 && apk add --no-cache ca-certificates

# copy the pre-built binary
COPY --from=builder /workspace/octopilot /usr/local/bin/octopilot

ENTRYPOINT ["octopilot"]
