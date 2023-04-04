package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	pb "rpc_test/protos/gpt"
	"time"

	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 50051, "port")
)

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) GetGPTMessage(ctx context.Context, in *pb.GPTRequest) (*pb.GPTReply, error) {
	return &pb.GPTReply{Message: "gpt response"}, nil
}

func (s *server) GetGPTStreamData(in *pb.GPTRequest, gptStream pb.Greeter_GetGPTStreamDataServer) error {
	log.Printf("GetGPTStreamData Request: %v", in.GetContent())
	messages := []string{
		"春眠不觉晓",
		"处处闻啼鸟",
		"夜来风雨声",
		"花落知多少",
	}

	log.Println("Send reply:")
	for _, msg := range messages {
		// 发送流式数据到客户端
		if err := gptStream.Send(&pb.GPTReply{
			Message: msg,
		}); err != nil {
			log.Printf("Send error, %v", err)
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func main() {
	flag.Parse()
	list, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("listen failed, %v", err)
	}
	s := grpc.NewServer()
	// 服务注册
	pb.RegisterGreeterServer(s, &server{})
	log.Printf("listen success, %v", list.Addr())
	if err := s.Serve(list); err != nil {
		log.Fatalf("rpc_server failed, %v", err)
	}
}
