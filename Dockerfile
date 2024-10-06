FROM arm64v8/golang:latest as build

WORKDIR /app

ENV PORT 1111
ENV GOOS linux
ENV GOARCH arm64
ENV BINARY bin/urlshortener

COPY go.mod .
COPY go.sum .
COPY main.go .

RUN go mod download

RUN go build -o $BINARY

CMD ["sh", "-c", "$BINARY"]