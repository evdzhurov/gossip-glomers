# gossip-glomers
A series of distributed systems challenges brought by Fly.io.

# Sources
 - https://fly.io/dist-sys/

# Setup Maelstrom (Jepsen-based distributed system testing framework)
 - https://github.com/jepsen-io/maelstrom/blob/main/README.md

# Build test executables
```
go build -o bin/echo ./cmd/echo
```

# Run Maelstrom tests
```
./scripts/01-echo.sh
```

