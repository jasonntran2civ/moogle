# syntax=docker/dockerfile:1.7
# Multi-stage Dockerfile for ingester-pubmed.
# Build context expected: the `ingest/` directory (so the shared go.mod
# is reachable). Set in deploy-cloud-run.yml.

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/ingester-pubmed ./cmd/ingester-pubmed

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/ingester-pubmed /ingester-pubmed
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/ingester-pubmed"]
