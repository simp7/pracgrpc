package main

import (
	"github.com/simp7/pracgrpc/model"
	pb "github.com/simp7/pracgrpc/model/ecommerce"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"time"
)

const (
	port          = ":50051"
	secretKey     = "secret"
	tokenDuration = 15 * time.Minute
)

func createUser(userStore model.UserStore, username, password, role string) error {
	user, err := model.NewUser(username, password, role)
	if err != nil {
		return err
	}
	return userStore.Save(user)
}

func seedUsers(userStore model.UserStore) error {
	err := createUser(userStore, "admin1", "secret", "admin")
	if err != nil {
		return err
	}
	return createUser(userStore, "user1", "secret", "user")
}

func accessibleRoles() map[string][]string {
	return map[string][]string{
		"/ecommerce.ProductInfo/addProduct":      {"admin"},
		"/ecommerce.OrderManagement/createOrder": {"admin", "user"},
	}
}

func main() {
	userStore := model.NewInMemoryUserStore()
	jwtManager := NewJWTManager(secretKey, tokenDuration)

	if err := seedUsers(userStore); err != nil {
		log.Fatal("cannot seed users: ", err)
	} else {
		log.Println("seed users successfully")
	}

	authServer := NewAuthServer(userStore, jwtManager)
	interceptor := NewAuthInterceptor(jwtManager, accessibleRoles())

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
	}

	s := grpc.NewServer(opts...)

	pb.RegisterProductInfoServer(s, &server{})
	pb.RegisterOrderManagementServer(s, &server{})
	pb.RegisterAuthServiceServer(s, authServer)
	reflection.Register(s)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
