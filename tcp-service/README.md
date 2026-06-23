**TCP service**

A small custom application-layer protocol built on TCP.

**Protocol**

Transport: TCP
Port: `9000`
Encoding: UTF-8
Format: one command per line, terminated by `\n`

```text
COMMAND [argument]\n
```

**Connection policy**

The client opens a TCP connection and sends one command.
The server validates the command, sends one response, then closes the connection.

The client opens a new TCP connection for every command.

**Commands**

`PING`

```text
PING\n
→ PONG\n
```

`PING` does not accept an argument.

`ECHO <text>`

```text
ECHO hello world\n
→ ECHO hello world\n
```

`ECHO` requires an argument.

**Errors**

Unknown command:

```text
FAKECOMMAND hello
→ ERROR: unknown command
```

Invalid request:

```text
PING hello
→ ERROR: invalid request

ECHO
→ ERROR: invalid request
```

**Internal networking**

The service is only available on the internal Docker `backend` network.

It is not exposed through a host port or Nginx.

Other containers connect to it using Docker Compose DNS:

```text
tcp-service:9000
```
