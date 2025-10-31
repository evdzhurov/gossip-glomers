package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

// When we learn about a new message we want
// to propagate it as a gossip to our peers.
// Keep track of the 'from' node (or client)
// so we don't gossip back to the origin.
type gossipEvent struct {
	from string
	msg  int
}

func main() {
	n := maelstrom.NewNode()

	msgs := make(map[int]struct{})
	msgs_mtx := sync.Mutex{}

	var peers = []string{}
	peers_mtx := sync.Mutex{}

	gossip_queue := make(chan gossipEvent, 128)
	defer close(gossip_queue)

	go func() {
		for ev := range gossip_queue {
			var local_peers = []string{}

			peers_mtx.Lock() // TODO: This can be a shared mutex

			for _, peer_id := range peers {
				if peer_id != ev.from {
					local_peers = append(local_peers, peer_id)
				}
			}

			peers_mtx.Unlock()

			for _, peer_id := range local_peers {
				err := n.Send(peer_id, map[string]any{
					"type":    "broadcast",
					"message": ev.msg,
				})

				if err != nil {
					fmt.Fprintf(os.Stderr, "gossip send to %v failed with err %v\n", peer_id, err)
				}
			}
		}
	}()

	n.Handle("broadcast", func(msg maelstrom.Message) error {

		type messageBody struct {
			Message int `json:"message"`
		}

		var body messageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		msgs_mtx.Lock()

		// Using a "set" in order to avoid duplicates
		_, seen := msgs[body.Message]

		if !seen {
			msgs[body.Message] = struct{}{}
		}

		msgs_mtx.Unlock()

		reply := map[string]any{
			"type": "broadcast_ok",
		}

		if !seen {
			gossip_queue <- gossipEvent{from: msg.Src, msg: body.Message}
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

		msgs_mtx.Lock()

		for v := range msgs {
			local_msgs = append(local_msgs, v)
		}

		msgs_mtx.Unlock()

		body["messages"] = local_msgs

		return n.Reply(msg, body)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {

		type topologyBody struct {
			Type     string              `json:"type"`
			Topology map[string][]string `json:"topology"`
		}

		var body topologyBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		local_peers, ok := body.Topology[n.ID()]
		if !ok {
			return fmt.Errorf("no entry for node %s in topology", n.ID())
		}

		peers_mtx.Lock()

		peers = slices.Clone(local_peers)

		peers_mtx.Unlock()

		reply := map[string]any{
			"type": "topology_ok",
		}

		return n.Reply(msg, reply)
	})

	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		// Ignore responses from gossip broadcasts to peers
		return nil
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
