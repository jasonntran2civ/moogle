module github.com/evidencelens/evidencelens/ingest

go 1.23

require (
	cloud.google.com/go/pubsub v1.45.1
	github.com/aws/aws-sdk-go-v2 v1.32.6
	github.com/aws/aws-sdk-go-v2/config v1.28.6
	github.com/aws/aws-sdk-go-v2/credentials v1.17.47
	github.com/aws/aws-sdk-go-v2/service/s3 v1.71.0
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/jackc/pgx/v5 v5.7.1
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.32.0
	go.opentelemetry.io/otel/sdk v1.32.0
	go.opentelemetry.io/otel/trace v1.32.0
	google.golang.org/protobuf v1.35.2
)
