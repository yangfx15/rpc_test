package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	pb "rpc_test/protos/gpt"
	"time"
)

const defaultName = "world"

var (
	addr = flag.String("addr", "localhost:50051", "")
	name = flag.String("name", defaultName, "")
)

func main() {
	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Dial failed, %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	steam, err := c.GetGPTStreamData(ctx, &pb.GPTRequest{
		Content: "背一下古诗《春眠》",
	})
	if err != nil {
		log.Fatalf("GetGPTMessage failed, %v", err)
	}
	log.Println("Get reply:")
	for {
		res, err := steam.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Recv failed, %v", err)
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("%v, %v\n", now, res.GetMessage())
	}
}
