// Copyright 2018 acrazing <joking.young@gmail.com>. All rights reserved.
// Since 2018-05-24 17:00:56
// Version 1.0.0
// Desc main.go
package main

import (
	"context"
	"log"
	"net"
	"time"

	".."
	"google.golang.org/grpc"
)

func main() {
	addr := "127.0.0.1:1234"

	// server
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer()
	uno.RegisterUnoServer(server, uno.Service)
	uno.Service.Init(&uno.Options{
		MinValue: 2,
		MaxValue: 5,
		TTF:      time.Second,
		TTL:      time.Second,
	})
	go uno.Service.Run(context.Background())
	go server.Serve(listener)

	// client
	conn, _ := grpc.Dial(addr, grpc.WithInsecure())
	client := uno.NewUnoClient(conn)
	no, err := client.Rent(context.Background(), &uno.Empty{})
	log.Printf("rent: %v, err: %v", no, err)
	time.Sleep(time.Second)
	_, err = client.Relet(context.Background(), no)
	log.Printf("relet err: %v", err)
	time.Sleep(time.Second)
	_, err = client.Relet(context.Background(), &uno.UnoMessage{No: 1})
	log.Printf("relet not exists err: %v", err)
	time.Sleep(time.Second)
	_, err = client.Return(context.Background(), no)
	log.Printf("return err: %v", err)
	time.Sleep(time.Second * 3)
	for i := uno.Service.MinValue - 1; i < uno.Service.MaxValue; i++ {
		no, err = client.Rent(context.Background(), &uno.Empty{})
		// the last one should be exhausted
		log.Printf("rent: %v, err: %v", no, err)
	}
}
