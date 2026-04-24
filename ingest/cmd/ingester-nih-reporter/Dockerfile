# syntax=docker/dockerfile:1.7
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"     -o /out/ingester-nih-reporter ./cmd/ingester-nih-reporter

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/ingester-nih-reporter /ingester-nih-reporter
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/ingester-nih-reporter"]
