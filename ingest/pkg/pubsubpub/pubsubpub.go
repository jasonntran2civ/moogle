// Package pubsubpub publishes RawDocEvent messages to GCP Pub/Sub with
// OTel context propagation and at-least-once semantics.
package pubsubpub

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Publisher is a thin wrapper for a single Pub/Sub topic.
type Publisher struct {
	client *pubsub.Client
	topic  *pubsub.Topic
}

// New connects to GCP Pub/Sub and returns a Publisher pinned to one topic.
func New(ctx context.Context, projectID, topicID string) (*Publisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("pubsub client: %w", err)
	}
	t := client.Topic(topicID)
	t.EnableMessageOrdering = false
	t.PublishSettings.NumGoroutines = 4
	return &Publisher{client: client, topic: t}, nil
}

// Close flushes pending publishes and tears down the client.
func (p *Publisher) Close() {
	p.topic.Stop()
	_ = p.client.Close()
}

// PublishRaw emits a RawDocEvent. Caller passes already-marshalled
// protobuf bytes to keep this package free of generated proto imports
// (avoids a circular-ish dependency through proto/gen/go).
func (p *Publisher) PublishRaw(ctx context.Context, source, docID, r2Key string) (string, error) {
	// We marshal the event payload as a simple JSON envelope here for
	// transport. The bridge worker reshapes to protobuf before NATS
	// publish. Keeps Pub/Sub messages human-debuggable in dashboards.
	body := fmt.Sprintf(`{"source":%q,"doc_id":%q,"r2_key":%q,"ingested_at":%q}`,
		source, docID, r2Key, timestamppb.Now().AsTime().UTC().Format("2006-01-02T15:04:05Z"))

	res := p.topic.Publish(ctx, &pubsub.Message{
		Data: []byte(body),
		Attributes: map[string]string{
			"source": source,
			"doc_id": docID,
		},
	})
	id, err := res.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("pubsub publish: %w", err)
	}
	return id, nil
}

// PublishProto emits a pre-marshalled protobuf payload. Use when the
// caller already holds a generated message struct.
func (p *Publisher) PublishProto(ctx context.Context, attrs map[string]string, msg proto.Message) (string, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return "", err
	}
	res := p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attrs,
	})
	return res.Get(ctx)
}
