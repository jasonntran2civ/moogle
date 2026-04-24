# syntax=docker/dockerfile:1.7
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"     -o /out/ingester-trials ./cmd/ingester-trials

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/ingester-trials /ingester-trials
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/ingester-trials"]
