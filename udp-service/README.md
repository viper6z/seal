**UDP live text relay**

A small custom application-layer protocol built on UDP.

Clients keep a local `current_text` value and send the latest version to the server. The server relays the newest text to all subscribed listeners, including the sender.

**Protocol**

Transport: UDP
Port: `9001`
Encoding: UTF-8
Format: one command per UDP datagram

Unlike TCP, messages do not need `\n` framing. One UDP datagram is already one message.

**Joining**

A client sends this when it starts:

```text
JOIN
```

The server stores the client address as a subscribed listener.

**Updating text**

The client sends its full current text:

```text
UPDATE hello world
```

The server increases its own sequence number and broadcasts the update to every subscribed client:

```text
TEXT 17 hello world
```

`17` is the server sequence number.

Listeners only display a `TEXT` message when its sequence number is newer than the one they have already seen.

**Client behaviour**

The client reads keypresses and updates its local `current_text`.

Every 0.1 seconds, it sends an `UPDATE` message only when the text has changed.

The client also listens for `TEXT` broadcasts from the server and redraws the latest received text.

**Server behaviour**

The server receives `JOIN` and stores the sender address as a subscriber.

When it receives `UPDATE <text>`, it creates a new server sequence number and broadcasts:

```text
TEXT <sequence> <text>
```

to all subscribed clients.

**Limits**

UDP does not guarantee delivery or ordering.

A missing update is fine because the next update contains the full current text.

This is a small real-time text relay, not a collaborative editor. If two clients type at once, the newest server update wins.

**Internal networking**

The service is only available on the internal Docker `backend` network.

It is not exposed through a host port or Nginx.

Other containers connect using Docker Compose DNS:

```text
udp-service:9001
```

