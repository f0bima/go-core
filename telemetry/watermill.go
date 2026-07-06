package telemetry

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// InjectWatermillContext injects the current OpenTelemetry trace context from ctx 
// into the Watermill message metadata.
func InjectWatermillContext(ctx context.Context, msg *message.Message) {
	if msg.Metadata == nil {
		msg.Metadata = make(message.Metadata)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(msg.Metadata))
}

// ExtractWatermillContext extracts the OpenTelemetry trace context from the 
// Watermill message metadata and returns a new context containing it.
func ExtractWatermillContext(ctx context.Context, msg *message.Message) context.Context {
	if msg.Metadata == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(msg.Metadata))
}

// TracingPublisherDecorator wraps a message.Publisher and creates an OpenTelemetry span
// for every published message.
type TracingPublisherDecorator struct {
	publisher message.Publisher
	tracer    trace.Tracer
}

func NewTracingPublisherDecorator(publisher message.Publisher, tracerName string) message.Publisher {
	return &TracingPublisherDecorator{
		publisher: publisher,
		tracer:    otel.Tracer(tracerName),
	}
}

func (d *TracingPublisherDecorator) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		// Extract context from metadata first, in case the message was deserialized 
		// (e.g. from SQL Outbox) and msg.Context() is empty.
		ctx := ExtractWatermillContext(msg.Context(), msg)
		
		ctx, span := d.tracer.Start(ctx, "AMQP Publish: "+topic, trace.WithSpanKind(trace.SpanKindProducer))
		
		InjectWatermillContext(ctx, msg)
		msg.SetContext(ctx)
		span.End()
	}
	return d.publisher.Publish(topic, messages...)
}

func (d *TracingPublisherDecorator) Close() error {
	return d.publisher.Close()
}

// TracingSubscriberDecorator wraps a message.Subscriber and creates an OpenTelemetry span
// for every received message.
type TracingSubscriberDecorator struct {
	subscriber message.Subscriber
	tracer     trace.Tracer
}

func NewTracingSubscriberDecorator(subscriber message.Subscriber, tracerName string) message.Subscriber {
	return &TracingSubscriberDecorator{
		subscriber: subscriber,
		tracer:     otel.Tracer(tracerName),
	}
}

func (d *TracingSubscriberDecorator) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	out, err := d.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return nil, err
	}

	tracedOut := make(chan *message.Message)
	go func() {
		defer close(tracedOut)
		for msg := range out {
			msgCtx := ExtractWatermillContext(msg.Context(), msg)
			msgCtx, span := d.tracer.Start(msgCtx, "AMQP Consume: "+topic, trace.WithSpanKind(trace.SpanKindConsumer))
			
			// Inject the new span back into metadata so the next handler can extract it
			InjectWatermillContext(msgCtx, msg)
			msg.SetContext(msgCtx)
			
			// We can't know when the message is acked/nacked to close the span precisely here
			// if we just pass the message. So we close it immediately or wrap the Ack.
			// For simplicity and less noise, we just close it after creation, 
			// and let the actual Handler create its own span.
			span.End()
			
			tracedOut <- msg
		}
	}()

	return tracedOut, nil
}

func (d *TracingSubscriberDecorator) Close() error {
	return d.subscriber.Close()
}
