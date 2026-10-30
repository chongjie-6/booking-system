package mq

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
)

// capture swaps produce for one that records which topic each write went to, for the length of the test.
func capture(t *testing.T) *[]string {
	var topics []string
	old := produce
	produce = func(_ context.Context, r *kgo.Record) error { topics = append(topics, r.Topic); return nil }
	t.Cleanup(func() { produce = old })
	return &topics
}

// TestHandleDispatch checks that handle routes an event to the handler for its type, decoded, and dead-letters
// unknown types and bad JSON without running a handler.
func TestHandleDispatch(t *testing.T) {
	sent := capture(t)
	type ev struct{ Type, UserID string }
	var got ev
	c := newConsumer("g", "t", map[string]Handler{
		"a": func(_ context.Context, b []byte) error { return json.Unmarshal(b, &got) },
	})

	c.handle(&kgo.Record{Value: []byte(`{"type":"a","userID":"u1"}`)}, 0)
	if got.UserID != "u1" || len(*sent) != 0 {
		t.Fatalf("handler for %q got %+v and forwarded to %v, want userID u1 and nothing forwarded", "a", got, *sent)
	}

	got = ev{}
	c.handle(&kgo.Record{Value: []byte(`{"type":"b","userID":"u2"}`)}, 0)
	c.handle(&kgo.Record{Value: []byte(`not json`)}, 0)
	if got != (ev{}) {
		t.Fatalf("unknown type or bad JSON reached a handler: %+v", got)
	}
	if want := []string{"t.g.dlq", "t.g.dlq"}; !slices.Equal(*sent, want) {
		t.Fatalf("forwarded to %v, want %v", *sent, want)
	}
}

// TestHandleRetries checks that an event failing at every stage moves through each retry topic, then the dead-letter topic.
func TestHandleRetries(t *testing.T) {
	sent := capture(t)
	c := newConsumer("g", "t", map[string]Handler{"a": func(context.Context, []byte) error { return errors.New("down") }})
	for stage := range retries + 1 {
		c.handle(&kgo.Record{Value: []byte(`{"type":"a"}`)}, stage)
	}
	if want := []string{"t.g.retry-1", "t.g.retry-2", "t.g.retry-3", "t.g.dlq"}; !slices.Equal(*sent, want) {
		t.Fatalf("forwarded to %v, want %v", *sent, want)
	}
}
