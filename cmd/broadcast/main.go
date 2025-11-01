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

type broadcastServer struct {
	node *maelstrom.Node

	msgs    map[int]struct{}
	msgsMtx sync.Mutex

	peers    []string
	peersMtx sync.Mutex

	gossipQueue chan gossipEvent
}

func newBroadcastServer(n *maelstrom.Node) *broadcastServer {
	s := &broadcastServer{
		node:        n,
		msgs:        make(map[int]struct{}),
		gossipQueue: make(chan gossipEvent, 128),
	}

	go s.gossipLoop()
	return s
}

func (s *broadcastServer) gossipLoop() {
	type pendingGossip struct {
		msg  int
		acks int
	}
	pending := make(map[int]*pendingGossip)

	for ev := range s.gossipQueue {
		var local_peers = []string{}

		s.peersMtx.Lock() // TODO: This can be a shared mutex
		for _, peer_id := range s.peers {
			if peer_id != ev.from {
				local_peers = append(local_peers, peer_id)
			}
		}
		s.peersMtx.Unlock()

		pending[ev.msg] = &pendingGossip{
			msg:  ev.msg,
			acks: len(local_peers),
		}

		for _, peer_id := range local_peers {
			err := s.node.Send(peer_id, map[string]any{
				"type":    "gossip",
				"message": ev.msg,
			})

			if err != nil {
				fmt.Fprintf(os.Stderr, "gossip sent to %v failed with err %v\n", peer_id, err)
			}
		}

		// TODO: Delete only when acknowledged
		delete(pending, ev.msg)
	}
}

func (s *broadcastServer) addUniqueMessage(src string, msg int) {

	s.msgsMtx.Lock()

	// Using a "set" in order to avoid duplicates
	_, seen := s.msgs[msg]

	if !seen {
		s.msgs[msg] = struct{}{}
	}

	s.msgsMtx.Unlock()

	if !seen {
		s.gossipQueue <- gossipEvent{from: src, msg: msg}
	}
}

func (s *broadcastServer) handleBroadcast(msg maelstrom.Message) error {

	type messageBody struct {
		Message int `json:"message"`
	}

	var body messageBody
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.addUniqueMessage(msg.Src, body.Message)

	reply := map[string]any{
		"type": "broadcast_ok",
	}

	return s.node.Reply(msg, reply)
}

func (s *broadcastServer) handleGossip(msg maelstrom.Message) error {

	type gossipBody struct {
		Message int `json:"message"`
	}

	var body gossipBody
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.addUniqueMessage(msg.Src, body.Message)

	reply := map[string]any{
		"type":    "gossip_ok",
		"message": body.Message,
	}

	return s.node.Reply(msg, reply)
}

func (s *broadcastServer) handleRead(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "read_ok"

	var local_msgs = []int{}

	s.msgsMtx.Lock()

	for v := range s.msgs {
		local_msgs = append(local_msgs, v)
	}

	s.msgsMtx.Unlock()

	body["messages"] = local_msgs

	return s.node.Reply(msg, body)
}

func (s *broadcastServer) handleTopology(msg maelstrom.Message) error {
	type topologyBody struct {
		Type     string              `json:"type"`
		Topology map[string][]string `json:"topology"`
	}

	var body topologyBody
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	local_peers, ok := body.Topology[s.node.ID()]
	if !ok {
		return fmt.Errorf("no entry for node %s in topology", s.node.ID())
	}

	s.peersMtx.Lock()
	s.peers = slices.Clone(local_peers)
	s.peersMtx.Unlock()

	reply := map[string]any{
		"type": "topology_ok",
	}

	return s.node.Reply(msg, reply)
}

func (s *broadcastServer) handleGossipOk(msg maelstrom.Message) error {

	type gossipOkBody struct {
		Message int `json:"message"`
	}

	var body gossipOkBody
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Handling gossip_ok for %v from %v\n", body.Message, msg.Src)
	return nil
}

func main() {
	n := maelstrom.NewNode()
	server := newBroadcastServer(n)

	n.Handle("broadcast", server.handleBroadcast)
	n.Handle("gossip", server.handleGossip)
	n.Handle("read", server.handleRead)
	n.Handle("topology", server.handleTopology)
	n.Handle("gossip_ok", server.handleGossipOk)

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
