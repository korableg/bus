# BUS: The Kafka event bus framework

![coverage](https://raw.githubusercontent.com/korableg/bus/badges/.badges/main/coverage.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/korableg/bus)](https://goreportcard.com/report/github.com/korableg/bus)
[![Go Reference](https://pkg.go.dev/badge/github.com/korableg/bus.svg)](https://pkg.go.dev/github.com/korableg/bus)

<p align="center">
<img src="./asset/gopher-ride-bus-circled.png" alt="Bus gopher logo" width="600" height="600"/>
</p>

This framework gives you a convenient mechanism for using Kafka. You do not need more cycles and care for commits in your code, just connect and use.

Some libraries are used under the hood, but they are not exported directly, their own abstractions are used instead.

To date, libraries are used:

- https://github.com/IBM/sarama for interact with Kafka
- https://github.com/cenkalti/backoff for handler retrying

## Examples

```bash
go get github.com/korableg/bus
```

This repository has a predefined codec [wire](./codec/proto/wire/). You may use this or create your own.

To use the wire codec you have to create event contracts proto file which should [use grpc options](./codec/proto/event/event.proto) with a topic and a key fields.
See examples in the [test](./test/test.proto) package.

### Producer

```go
p, err := producer.New(wireCodec.Encoder(), brokers)
if err != nil {
    return err
}

// Async sending
err = p.Send(ctx, &events.YourEvent{})
if err != nil {
    return err
}

// Sync sending
err = p.SendSync(ctx, &events.YourEvent{})
if err != nil {
    return err
}

// You have to close producer before closing your application to prevent lose of the messages in the buffer
err = p.Close()
if err != nil {
    return err
}
```

### Consumer

```go
c, err := consumer.NewGroup(brokers, "test")
if err != nil {
    return err
}

// Close the group before closing your application
defer c.Close()

err = c.AddHandler(consumer.NewHandler(wireCodec.Decoder[*events.YourEvent](),
    func(ctx context.Context, msg *events.YourEvent, raw consumer.Message) error {
        // handle your message
        return nil
    }))
if err != nil {
    return err
}

err = c.AddHandler(consumer.NewHandlerBatch(wireCodec.Decoder[*events.YourEvent](),
    func(ctx context.Context, msgs []*events.YourEvent, raws []consumer.Message) error {
        // handle your messages
        return nil
    }))
if err != nil {
    return err
}

// Run blocks until the context is canceled or the group is closed
err = c.Run(ctx)
if err != nil {
    return err
}
```

With the wire codec the decoder may be omitted: [protobus](./protobus/) infers the event type from the handler function.

```go
err = c.AddHandler(protobus.NewHandler(
    func(ctx context.Context, msg *events.YourEvent, raw consumer.Message) error {
        // handle your message
        return nil
    }))

err = c.AddHandler(protobus.NewHandlerBatch(
    func(ctx context.Context, msgs []*events.YourEvent, raws []consumer.Message) error {
        // handle your messages
        return nil
    }))
```

The batch handler keeps one buffer for all the partitions of the topic, so a batch may contain the messages of several partitions.
The calls of the batch function are sequential.
A batch is flushed when it reaches the batch size or when no message has arrived for the flush timeout: while the messages keep coming the batch waits for its size, and the timeout keeps them from getting stale when the flow slows down.

### Options

[Producer](./producer/opts.go), [consumer](./consumer/opts.go) and [handler](./consumer/handler_opts.go) have an options to manipulate them behavior.

#### Authentication and TLS

Producer and consumer accept the same connection options:

```go
// SASL PLAIN, SCRAM-SHA-256 and SCRAM-SHA-512
producer.WithSASL(sasl.TypeSCRAMSHA512, "user", "password")
consumer.WithSASL(sasl.TypeSCRAMSHA256, "user", "password")

// SASL OAUTHBEARER: implement sasl.TokenProvider (it should cache and refresh the token)
producer.WithSASLOAuth(tokenProvider)
consumer.WithSASLOAuth(tokenProvider)

// TLS (nil means the default tls.Config)
producer.WithTLS(&tls.Config{})
consumer.WithTLS(&tls.Config{})
```

SASL PLAIN sends the password in plain text, use it together with `WithTLS`.

#### Asynchronous delivery errors

`Send` returns as soon as the message is accepted by the buffer, so the delivery errors are reported later. Besides the log, you may set a handler:

```go
p, err := producer.New(wireCodec.Encoder(), brokers, producer.WithErrorHandler(func(err error) {
    var sendErr *producer.SendError
    if errors.As(err, &sendErr) {
        // sendErr.Topic, sendErr.Key, sendErr.ID, sendErr.Type, sendErr.Msg (your message), sendErr.Err (the cause)
    }
}))
```

The handler is called from the producer's event loop: it must not block, otherwise the delivery of the other messages (and `Close`) stalls.
It is also called for the errors which happen while the producer is closing.
The wire codec rejects a nil message (a typed nil pointer included): `Send` and `SendSync` return `codec.ErrNilMessage`. `Send` stops waiting for a free buffer slot when the context is done.

### Commit policy

The message is considered processed after exiting the handler, regardless of whether it was processed with an error or not.  
The single message handler resets the offset to commit immediately after exiting the function. The batch handler does not transmit offsets until it flushes the buffer of accumulated messages.  
A message of another type (or a broken one) is skipped and counts as handled: the batch handler advances its offset at once, or after the flush if the buffer holds messages of that partition.  
If the group has several handlers on the same topic, then the minimum possible offset from all handlers is committed.

### Retry policy

By default, all handlers have endlessly trying to process the message until err == nil.
You may change this behavior by setting the handler options or [mark error](./consumer/error.go#L23) as a permanent.

### Benchmarks

```bash
make bench                         # starts Kafka from docker-compose.yml, runs the benchmarks, stops Kafka
make bench BENCH_FLAGS=-count=3    # extra go test flags
```

The numbers below are the medians of 3 runs on Apple M4 Pro (14 cores), macOS, Go 1.27.3, sarama v1.61.1
and a single `apache/kafka:4.4.0` broker in Docker (replication factor 1). The options are the defaults:
the producer is idempotent and waits for all in-sync replicas with one in-flight request per connection,
the group commits every 20 messages or every second. The only exception is the flush timeout of the batch handler in the Kafka benchmark:
it is 10 ms, so the last incomplete batch doesn't wait the default 10 s. The handler functions do nothing.

Kafka, end to end ([test/wire/bench_test.go](./test/wire/bench_test.go)), a message is `TestEvent1` with a ~30-byte payload:

| Benchmark | Throughput | Per message |
|---|---:|---:|
| `Send`, until the broker has acknowledged all the messages | 486k msg/s | 2.1 µs |
| `SendSync`, one after another | 5.4k msg/s | 184 µs |
| `SendSync` from 32 goroutines | 36k msg/s | 28 µs |
| Group with `NewHandler`, 3 partitions | 535k msg/s | 1.9 µs |
| Group with `NewHandlerBatch` (batch of 100), 3 partitions | 684k msg/s | 1.5 µs |

The consumer timer runs from the first handled message to the last one, so producing the messages and joining the group are not counted.
The broker runs in a Docker VM on the same machine: take the numbers as the order of magnitude, not as the capacity of a production cluster.

The cost of the library itself, without Kafka:

| Benchmark | Time | Memory | Allocations |
|---|---:|---:|---:|
| [wire encoder](./codec/proto/wire/bench_test.go): marshal, topic, type, key, headers | 263 ns | 256 B | 7 |
| [wire decoder](./codec/proto/wire/bench_test.go): unmarshal | 139 ns | 224 B | 5 |
| [Producer](./producer/bench_test.go): the Kafka message (encoding, key, UUIDv7 ID, headers) | 670 ns | 928 B | 17 |
| [`NewHandler`](./consumer/bench_test.go): one message (decoding, timeout, retry, metrics, offset) | 707 ns | 1064 B | 17 |
| [`NewHandlerBatch`](./consumer/bench_test.go): one message, the batch call amortized | 235 ns | 425 B | 5 |

### Contributing

You are welcome! ♥️