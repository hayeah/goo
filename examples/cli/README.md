
- `di.go` has all the wire providers.
  - also wire set for different injection setups
- `app.go` has the cli parsing and "top level" provider

look at Makefile for how to generate injection with wire, and how to build the app.