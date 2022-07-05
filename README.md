# Named-Chatroom
Golang and React practice.

Uses [Gorilla/websocket](https://github.com/gorilla/websocket) package, extends the chat example with Message objects containing the senders name and timestamp.

Adds a React front end.

# Running:
First build the react front end:
```bash
cd my-app
npm run build
```

Then `cd` back to the root of the repository and start the server with
```bash
go run .
```

## TODOs

- [ ] Add docker
