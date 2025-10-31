package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	msgs := make(map[int]struct{})
	mtx := sync.Mutex{}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Using a "set" in order to avoid duplicates
		val, ok := body["message"].(float64)
		if !ok {
			return fmt.Errorf("ignoring broadcast with non-number message")
		}

		mtx.Lock()

		msgs[int(val)] = struct{}{}

		mtx.Unlock()

		reply := map[string]any{
			"type": "broadcast_ok",
		}

		return n.Reply(msg, reply)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "read_ok"

		var local_msgs = []int{}

		mtx.Lock()

		for v := range msgs {
			local_msgs = append(local_msgs, v)
		}

		mtx.Unlock()

		body["messages"] = local_msgs

		return n.Reply(msg, body)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		reply := map[string]any{
			"type": "topology_ok",
		}

		return n.Reply(msg, reply)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
