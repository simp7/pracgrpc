package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net"
	"os"
	pb "productinfo/service/ecommerce"
	"time"
)

const (
	port    = ":50051"
	crtFile = "../server.crt"
	keyFile = "../server.key"
	caFile  = "../rootCA.crt"
)

func orderUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Println("=== [Server Interceptor]", info.FullMethod)
	m, err := handler(ctx, req)

	log.Printf("Post Proc Message: %s", m)

	return m, err
}

type wrappedStream struct {
	grpc.ServerStream
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("===== [Server Stream Intercepter Wrapper] "+"Receive a message (Type: %T) at %s", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf(" ===== [Server Stream Interceptor Wrapper] "+"Send a message (Type: %T) at %s", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.SendMsg(m)
}

func newWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s}
}

func orderServerStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("===== [Server Stream Interceptor] ", info.FullMethod)
	err := handler(srv, newWrappedStream(ss))
	if err != nil {
		log.Printf("RPC failed with error %v", err)
	}
	return err
}

func main() {
	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(caFile)
	if err != nil {
		log.Fatalf("could not read ca certificate: %s", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("failed to append ca certificate")
	}
	mtls := tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, Certificates: []tls.Certificate{cert}, ClientCAs: certPool}
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(orderUnaryServerInterceptor),
		grpc.StreamInterceptor(orderServerStreamInterceptor),
		grpc.Creds(credentials.NewTLS(&mtls)),
	}

	s := grpc.NewServer(opts...)

	pb.RegisterProductInfoServer(s, &server{})
	pb.RegisterOrderManagementServer(s, &server{})

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
