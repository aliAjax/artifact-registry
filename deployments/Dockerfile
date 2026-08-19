FROM golang:1.23 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/registry ./cmd/registry
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/registry /registry
ENTRYPOINT ["/registry"]
