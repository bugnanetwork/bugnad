ARG BGAPAD_IMAGE
ARG BGAPAMINER_IMAGE

FROM ${BGAPAD_IMAGE} as bugnad
FROM ${BGAPAMINER_IMAGE} as bugnaminer

FROM golang:1.19-alpine

RUN mkdir -p /go/src/github.com/bugnanetwork/bugnad

WORKDIR /go/src/github.com/bugnanetwork/bugnad

RUN apk add bash build-base git

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

COPY --from=bugnad /app/ /app/
COPY --from=bugnaminer /app/ /app/
ENV PATH="/app:${PATH}"

WORKDIR /go/src/github.com/bugnanetwork/bugnad/stability-tests

RUN go install ./...

ENTRYPOINT ["./run/run.sh"]