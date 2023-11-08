package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	pb "github.com/Juules32/Node/proto"
	"google.golang.org/grpc"
)

type node struct {
	pb.UnimplementedTokenRingServer
	peerAddress    string // Address of the peer node
	hasToken       bool   // Indicates if this node has the token
	mu             sync.Mutex
	wantsToPerform bool
}

func (n *node) PassToken(ctx context.Context, token *pb.Token) (*pb.Ack, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Process the token (critical section) when received.
	if token.HasToken {
		log.Printf("Node has received the token: %s", token.Data)
		// Perform critical section operations here.
		// ...

		if n.wantsToPerform {

			f, err := os.OpenFile("log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				log.Fatalf("error opening file: %v", err)
			}

			log.SetOutput(f)

			log.Println("Performed Transaction")
			f.Close()

			l := log.New(os.Stdout, "[AAA] ", 2)
			l.Printf("Data written to file successfully!")

			// Now pass the token back to the peer node.
			n.wantsToPerform = false
		}
		n.hasToken = true
		return &pb.Ack{Success: true}, nil
	}

	return &pb.Ack{Success: false}, nil
}

func (n *node) sendMessage(ctx context.Context, selfAddress string, peerAddress string) {
	for {
		if n.hasToken {
			n.mu.Lock()
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

			fmt.Println(selfAddress + " passed the token to " + peerAddress)

			// After sending the token, the node should wait for it to come back.
			n.hasToken = false
			n.mu.Unlock()
		}

		// Wait for some time before trying to send the token again.
		time.Sleep(100 * time.Millisecond)
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
	n := &node{peerAddress: peerAddress, hasToken: true, wantsToPerform: false} // Node starts with the token
	pb.RegisterTokenRingServer(grpcServer, n)

	go func() {
		for {
			n.wantsToPerform = true
			time.Sleep(time.Second * 10)
		}
	}()

	// Start a goroutine for sending messages when this node has the token
	go n.sendMessage(context.Background(), selfAddress, peerAddress)

	log.Printf("Node listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

// token_ring.proto
