FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS build
WORKDIR /src
COPY . .
RUN go mod download
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/location ./cmd/location

FROM alpine
COPY --from=build /out/location /bin/location
ENTRYPOINT ["/bin/location"]