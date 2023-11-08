package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/Juules32/Node/proto"
	"google.golang.org/grpc"
)

type node struct {
	pb.UnimplementedTokenRingServer
	peerAddress string // Address of the peer node
	hasToken    bool   // Indicates if this node has the token
	mu          sync.Mutex
}

func (n node) PassToken(ctx context.Context, token pb.Token) (pb.Ack, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Process the token (critical section) when received.
	if token.HasToken {
		log.Printf("Node has received the token: %s", token.Data)
		// Perform critical section operations here.
		// ...

		// Simulate work being done in the critical section.
		// time.Sleep(time.Duration(rand.Intn(5)) time.Second)

		// Now pass the token back to the peer node.
		n.hasToken = false
		return &pb.Ack{Success: true}, nil
	}

	return &pb.Ack{Success: false}, nil
}

func (n *node) sendMessage(ctx context.Context) {
	for {
		n.mu.Lock()
		if n.hasToken {
			// Perform actions as the client, sending the token to the peer.
			conn, err := grpc.Dial(n.peerAddress, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				log.Fatalf("did not connect: %v", err)
			}
			defer conn.Close()

			c := pb.NewTokenRingClient(conn)
			_, err = c.PassToken(ctx, &pb.Token{Data: "Your token data here", HasToken: true})
			if err != nil {
				log.Fatalf("could not pass token: %v", err)
			}

			// After sending the token, the node should wait for it to come back.
			n.hasToken = false
		}
		n.mu.Unlock()

		// Wait for some time before trying to send the token again.
		time.Sleep(1 * time.Second)
	}
}
func main() {
	// Configuration
	selfAddress := "localhost:50051" // Address of this node
	peerAddress := "localhost:50052" // Address of the peer node

	// Starting the server
	lis, err := net.Listen("tcp", selfAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	n := &node{peerAddress: peerAddress, hasToken: true} // Node starts with the token
	pb.RegisterTokenRingServer(grpcServer, n)

	// Start a goroutine for sending messages when this node has the token
	go n.sendMessage(context.Background())

	log.Printf("Node listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// token_ring.proto
