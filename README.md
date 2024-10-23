# BUS: The Kafka event bus framework

<img src="./asset/gopher-ride-bus.png" alt="Gopher Logo" style="display: block; margin: 0 auto; border-radius: 50%; max-width: 50%; padding: 10px;"/>

This framework gives you a convenient mechanism for using Kafka. You do not need more cycles and care for commits in your code, just connect and use.

Some libraries are used under the hood, but they are not exported directly, their own abstractions are used instead.

To date, libraries are used:

- https://github.com/IBM/sarama for interact with Kafka
- https://github.com/cenkalti/backoff for handler retrying

