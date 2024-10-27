# BUS: The Kafka event bus framework

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

err = cons.AddHandler(consumer.NewHandler(wireCodec.Decoder[*events.YourEvent](),
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
```

### Options

[Producer](./producer/opts.go), [consumer](./consumer/opts.go) and [handler](./consumer/handler_opts.go) have an options to manipulate them behavior.

### Commit policy

The message is considered processed after exiting the handler, regardless of whether it was processed with an error or not.  
The single message handler resets the offset to commit immediately after exiting the function. The batch handler does not transmit offsets until it flushes the buffer of accumulated messages.  
If the group has several handlers on the same topic, then the minimum possible offset from all handlers is committed.

### Retry policy

By default, all handlers have endlessly trying to process the message until err!= nil.
You may change this behavior by setting the handler options or [mark error](./consumer/error.go#L23) as a permanent.

### Contributing

You are welcome! ♥️