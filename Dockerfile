FROM golang:1.12.5 AS build
WORKDIR /micro-srv-test

ENV GOPROXY=https://gocenter.io
COPY go.mod ./
RUN go mod download

COPY main.go .
COPY greet/ greet/
RUN CGO_ENABLED=0 GOOS=linux GOFLAGS=-ldflags=-w go build -o /go/bin/micro-srv-test -ldflags=-s -v github.com/sel/micro-srv-test

FROM scratch AS final
COPY --from=build /go/bin/micro-srv-test /bin/micro-srv-test
