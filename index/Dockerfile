# syntax=docker/dockerfile:1.7
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/indexer ./cmd/indexer

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/indexer /indexer
USER nonroot:nonroot
ENTRYPOINT ["/indexer"]
