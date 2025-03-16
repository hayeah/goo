
- `di.go` has all the wire providers.
  - also wire set for different injection setups
- `app.go` has the cli parsing and "top level" provider
- note how InitMain returns constructs a goo.Main that does graceful shutdown. goo.Main requires a goo.Runner.
- note how App implements goo.Runner
- note: migration is hooked into App.Run

look at Makefile for how to generate injection with wire, and how to build the app.