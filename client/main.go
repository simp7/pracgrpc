package main

import (
	"context"
	pb "github.com/simp7/pracgrpc/model/ecommerce"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"log"
	"time"
)

const (
	address         = "localhost:50051"
	username        = "admin1"
	password        = "secret"
	refreshDuration = time.Second * 30
)

func authMethods() map[string]bool {
	return map[string]bool{
		"/ecommerce.ProductInfo/addProduct":      true,
		"/ecommerce.OrderManagement/createOrder": true,
	}
}

func main() {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}
	authClient := NewAuthClient(conn, username, password)

	interceptor, err := NewAuthInterceptor(authClient, authMethods(), refreshDuration)
	if err != nil {
		log.Fatal("cannot create auth interceptor: ", err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	}

	conn, err = grpc.Dial(address, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Fatal("Error when close connection: ", err)
		}
	}()

	c := pb.NewProductInfoClient(conn)
	orderClient := pb.NewOrderManagementClient(conn)

	clientDeadline := time.Now().Add(2 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), clientDeadline)
	defer cancel()

	r, err := c.AddProduct(ctx, &pb.Product{
		Name:        "Apple iPhone 12",
		Description: "Meet Apple iPhone 12. All-new dual-camera system with Ultra Wide and Night mode.",
		Price:       float32(1000.0),
	})
	if err != nil {
		log.Fatalf("error when adding prodduct: %v", err)
	}
	product, err := c.GetProduct(ctx, &pb.ProductID{Value: r.Value})
	product, err = c.GetProduct(ctx, &pb.ProductID{Value: product.Id})

	orderId, err := orderClient.CreateOrder(ctx, &pb.Order{
		Items:       []string{"Google glass"},
		Description: "Will be released?",
		Price:       100,
		Destination: "Seoul",
	})

	retrievedOrder, err := orderClient.GetOrder(ctx, wrapperspb.String(orderId.GetValue()))
	log.Print("GetOrder Response -> : ", retrievedOrder.String())

	searchStream, _ := orderClient.SearchOrders(ctx, wrapperspb.String("Google"))

	for {
		searchOrder, err := searchStream.Recv()
		if err == io.EOF {
			break
		}
		log.Print("Search Result: ", searchOrder)
	}

	updateStream, err := orderClient.UpdateOrders(ctx)

	updOrder1 := &pb.Order{
		Id:          "aaaa",
		Items:       []string{"Google glass"},
		Description: "Will be released?",
		Price:       100,
		Destination: "Seoul",
	}

	updOrder2 := &pb.Order{
		Id:          "fjdkao",
		Items:       []string{"iPhone 15 pro max"},
		Description: "Will be released?",
		Price:       100,
		Destination: "Seoul",
	}

	updOrder3 := &pb.Order{
		Id:          "fjdkao",
		Items:       []string{"iPhone 15 pro"},
		Description: "Will be released?",
		Price:       100,
		Destination: "Seoul",
	}

	updOrders := []*pb.Order{updOrder1, updOrder2, updOrder3}
	for _, v := range updOrders {
		if err = updateStream.Send(v); err != nil {
			log.Fatal("Error!")
		}
	}

	updateRes, err := updateStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v", updateStream, err)
	}
	log.Printf("Update Orders Res: %s", updateRes)

}
