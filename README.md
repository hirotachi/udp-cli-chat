# go-cli-udp-chat
Chat server and client written in go using the UDP protocol and backed by redis as a database.

## Usage

You can use `make` commands:

Build and run `udp-server`:

```bash
$ make run-server
```

Build and run `udp-client`:

```bash
$ make run-client
```


Build `udp-server`, `udp-client` and put binaries into corresponding `cmd/*` dir:

```bash
$ make build
```

Install `udp-server`, `udp-client` and put binaries into `$GOPATH/bin/` dir:

```bash
$ make install
```

## Server API Documentation

### Sending Packets
To interact with the `udp-server` certain commands must be sent through UDP connection with data:

`/connect>{LoginInput}` register client and get an assigned ID with history of last 20 entries.

```go
type LoginInput struct {
	Username string `json:"username"` // required
}
```


`/add_message>{NewMessage}` broadcast message to all online connected clients.
```go
type NewMessage struct {
	Content  string `json:"content"`   // required
	AuthorID string `json:"author_id"` // required (assigned id returned on InitialPayload after first connection) 
}
```


`/delete_message>{Message}` deletes message from db and broadcasts changes to all clients.
```go
type Message struct {
	ID        string    `json:"id"`         //required
	Content   string    `json:"content"`    //required
	AuthorID  string    `json:"author_id"`  //required
	CreatedAt time.Time `json:"created_at"` //required
	Edited    bool      `json:"edited"`     //required
}
```

`/disconnect>{ClientID}` disconnects client from chat.

```go
type ClientID string // required (assignedID from initialPayload)
```

### Receiving Packets
Receiving Packets from UDP connection will indicate how clients update chat:

`/initial_payload>{IntialPayload}` received on first connection with assignedID and history length to join by client.
```go
type InitialPayload struct {
	AssignedId    string `json:"assigned_id"`
	HistoryLength int    `json:"history_length"`
}
```


`/add_history>{HistoryLog}` multiple Packets received depending on history size to be joined with order after all history has been received to avoid packet loss.
```go
type HistoryLog struct {
	Order   int      `json:"order"`
	Message *Message `json:"message"`
}
```

`/delete_message>{MessageID}` received when a client deletes his message to be reflected on all clients chats.
```go
type MessageID string
```



