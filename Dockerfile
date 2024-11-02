# BUILD STAGE
FROM arm64v8/golang:latest AS build

WORKDIR /app

ENV PORT 1111
ENV GOOS linux
ENV GOARCH arm64
ENV BINARY /app/bin/urlshortener

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .

# Build the binary with static linking
RUN CGO_ENABLED=0 go build -o $BINARY

# RUNTIME STAGE
FROM arm64v8/alpine:latest

WORKDIR /app
ENV PORT 1111
ENV BINARY /app/bin/urlshortener

# Create a new user and group
RUN addgroup -S ramit && adduser -S ramit -G ramit

# Copy the binary from the build stage
COPY --from=build $BINARY $BINARY

# Set ownership and permissions for the binary
RUN chown ramit:ramit $BINARY && chmod +x $BINARY

# Switch to the user
USER ramit

# Set the command to run the binary
CMD ["sh", "-c", "$BINARY"]
