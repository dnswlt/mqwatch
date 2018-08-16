# mqwatch 

Binds a queue to one or more RabbitMQ topic exchanges and listens for messages for a given routing key.
Expects messages to be UTF-8 encoded JSON.

        Usage of C:\devsbb\go\bin\mqwatch.exe:
        -buf int
                Number of messages kept in memory (default 100000)
        -exchange string
                Exchange(s) to bind to (comma separated) (default "lenkung")
        -key string
                Routing key to use in queue binding (default "#")
        -maxresult int
                Max. number of messages returned for query (default 1000)
        -port int
                TCP port web UI (default 9090)
        -url string
                URL to connect to (default "amqp://localhost:5672/")

Open http://localhost:9090 and fire some queries.

## Query syntax

Queries are comma-separated, comma acts as an OR (not an AND). Each query consists of
one or more of the following filters:

* A range `#100-200` selecting messages with sequence numbers 100 through 200. Sequence
  numbers are generated on message receipt, starting at 0.
* Routing keys: `key:some.key` matches messages whose routing key contains `some.key`.
* Arbitrary text that must occur in the message. `foosel basel` matches messages that contain
  the string `foosel` and the string `basel`.
