# -- multistage docker build: stage #1: build stage
FROM golang:1.22.0-alpine AS build

RUN mkdir -p /go/src/github.com/bugnanetwork/bugnad

WORKDIR /go/src/github.com/bugnanetwork/bugnad

RUN apk add --no-cache curl git openssh binutils gcc musl-dev

COPY go.mod .
COPY go.sum .


# Cache bugnad dependencies
RUN go mod download

COPY . .
RUN mkdir -p /bugna/bin/
RUN go build $FLAGS -o /bugna/bin/ ./cmd/...

# --- multistage docker build: stage #2: runtime image
FROM alpine
WORKDIR /root/

RUN apk add --no-cache ca-certificates tini

COPY --from=build /bugna/bin/* /usr/bin/

ENTRYPOINT [ "/usr/bin/bugnad" ]
