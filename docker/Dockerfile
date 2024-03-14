# -- multistage docker build: stage #1: build stage
FROM golang:1.19-alpine AS build

RUN mkdir -p /go/src/github.com/bugnanetwork/bugnad

WORKDIR /go/src/github.com/bugnanetwork/bugnad

RUN apk add --no-cache curl git openssh binutils gcc musl-dev

COPY go.mod .
COPY go.sum .


# Cache bugnad dependencies
RUN go mod download

COPY . .

RUN go build $FLAGS -o bugnad .

# --- multistage docker build: stage #2: runtime image
FROM alpine
WORKDIR /app

RUN apk add --no-cache ca-certificates tini

COPY --from=build /go/src/github.com/bugnanetwork/bugnad/bugnad /app/
COPY --from=build /go/src/github.com/bugnanetwork/bugnad/infrastructure/config/sample-bugnad.conf /app/

USER nobody
ENTRYPOINT [ "/sbin/tini", "--" ]
