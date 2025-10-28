package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	var seq uint64 = 0
	var nodePrefix uint64 = 0

	n.Handle("init", func(msg maelstrom.Message) error {
		// Assuming Maelstrom calls our handler immediately after parsing "init"

		nodeNum, err := strconv.Atoi(strings.TrimPrefix(n.ID(), "n"))
		if err == nil {
			atomic.StoreUint64(&nodePrefix, uint64(nodeNum)<<48)
			fmt.Fprintf(os.Stderr, "Node prefix: %d\n", nodePrefix)
		}

		return nil
	})

	n.Handle("generate", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// We split the unique id into a 16 bit node prefix and a 48 bit seq id
		// Masking the 48-bit seq id in case of overflow - this is mostly cosmetic since we'll break unique id
		// generation anyway but seems better behaved than overrunning the prefix bits.

		// Although nodePrefix is set once before any generate message we're still using atomics
		// since the init handlers is concurrent with the generate handlers and the memory model
		// does not guarantee that the write will be propagated unless its atomic.

		next_id := atomic.LoadUint64(&nodePrefix) | (atomic.AddUint64(&seq, 1) & ((1 << 48) - 1))

		body["type"] = "generate_ok"
		body["id"] = next_id

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
