**My own application layer protocol**

Transport layer: TCP

Encoding: UTF-8

Format: one command per line ended with \n

Request format: COMMAND [argument]\n

Connection policy
- The client initiates a TCP connection and sends one command.
- The server responds.
- The server then closes the connection.

Commands/Requests:
- PING
- ECHO <text>
- INFO

Responses:
- PONG
- ECHO <text>
- INFO server=tcp-service timestamp=<ISO-8601 timestamp>

Errors
- ERROR unknown-command
- ERROR invalid-request



