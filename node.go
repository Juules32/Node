package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	pb "github.com/Juules32/Node/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type node struct {
	portNr         int
	nextPortNr     int  // Address of the peer node
	hasToken       bool // Indicates if this node has the token
	wantsToPerform bool
	mu             sync.Mutex
}

func (n *node) PassToken(ctx context.Context, token *pb.Token) (*emptypb.Empty, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.wantsToPerform {
		performActionInCriticalSection(n.portNr)
		n.wantsToPerform = false
	}

	n.hasToken = true

	return &emptypb.Empty{}, nil
}

func (n *node) dialNextInLine(ctx context.Context) {
	n.mu.Lock()
	defer n.mu.Unlock()

	//Find the peer address
	ports := readPorts()

	if len(ports) < 2 {
		writeToLogAndTerminal("Node with port " + strconv.Itoa(n.portNr) + " is all alone...")
		return
	}

	i := findIndex(ports, n.portNr)
	if i+1 == len(ports) {
		n.nextPortNr = ports[0]
	} else {
		n.nextPortNr = ports[i+1]
	}

	conn, err := grpc.Dial("localhost:"+strconv.Itoa(n.nextPortNr), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewTokenRingClient(conn)

	_, err = c.PassToken(ctx, &pb.Token{})
	if err != nil {
		log.Fatalf("could not pass token: %v", err)
	}

	n.hasToken = false

	writeToLogAndTerminal("Node with port " + strconv.Itoa(n.portNr) + " passed the token to Node with port " + strconv.Itoa(n.nextPortNr))

}

func main() {

	portNr := 50000
	ports := readPorts()

	f, err := os.OpenFile("ports.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	logger := log.New(f, "", 0)

	hasToken := false
	if len(ports) == 0 {
		hasToken = true
		logger.Println(portNr)
	} else {
		portNr = ports[len(ports)-1] + 1
		logger.Println(portNr)
	}

	f.Close()

	// Starting the server
	lis, err := net.Listen("tcp", "localhost:"+strconv.Itoa(portNr))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	n := &node{portNr: portNr, nextPortNr: -1000, hasToken: hasToken, wantsToPerform: false} // Node starts with the token
	pb.RegisterTokenRingServer(grpcServer, n)

	writeToLogAndTerminal("Node with port " + strconv.Itoa(n.portNr) + " entered the server localhost")

	go func() {
		for {
			n.wantsToPerform = true
			time.Sleep(time.Second * 10)
		}
	}()

	go func() {
		for {
			if n.hasToken {
				// Start a goroutine for sending messages when this node has the token
				n.dialNextInLine(context.Background())
			}
			// Wait for some time before trying to send the token again.
			time.Sleep(2 * time.Second)
		}
	}()

	log.Printf("Node listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Utility function to find the index of the node's port in the ports.txt file
func findIndex(slice []int, target int) int {
	for i, value := range slice {
		if value == target {
			return i
		}
	}
	return -1
}

// Utility function that reads and returns all the port numbers in the ports.txt file
func readPorts() []int {
	f, err := os.Open("ports.txt")
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	scanner := bufio.NewScanner(f)
	var ports []int
	for scanner.Scan() {
		port, err := strconv.Atoi(scanner.Text())
		if err != nil {
			log.Printf("error converting to integer: %v", err)
			continue
		}
		ports = append(ports, port)
	}
	f.Close()
	return ports
}

func performActionInCriticalSection(portNr int) {

	writeToLogAndTerminal("Node with port number " + strconv.Itoa(portNr) + " enters the critical section")

	writeToLogAndTerminal("Node with port number " + strconv.Itoa(portNr) + " performs action in critical section")

	writeToLogAndTerminal("Node with port number " + strconv.Itoa(portNr) + " leaves the critical section")

}

func writeToLogAndTerminal(message string) {
	fmt.Println(message)

	f, err := os.OpenFile("log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)

	log.Println(message)

	defer f.Close()
}
