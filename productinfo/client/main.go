package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"log"
	pb "productinfo/client/ecommerce"
	"time"
)

const (
	address = "localhost:50051"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := pb.NewProductInfoClient(conn)
	orderClient := pb.NewOrderManagementClient(conn)

	name := "Apple iPhone 12"
	description := `Meet Apple iPhone 12. All-new dual-camera system with Ultra Wide and Night mode.`
	price := float32(1000.0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.AddProduct(ctx, &pb.Product{Name: name, Description: description, Price: price})
	if err != nil {
		log.Fatalf("Could not add product: %v", err)
	}
	log.Printf("Product ID: %s added successfully", r.Value)

	product, err := c.GetProduct(ctx, &pb.ProductID{Value: r.Value})
	if err != nil {
		log.Fatalf("Could not get product: %v", err)
	}
	log.Print("Product: ", product.String())

	product, err = c.GetProduct(ctx, &pb.ProductID{Value: product.Id})
	if err != nil {
		log.Fatalf("Could not get product: %v", err)
	}
	log.Print("Product: ", product.String())

	orderId, err := orderClient.CreateOrder(ctx, &pb.Order{
		Items:       []string{"Google glass"},
		Description: "Will be released?",
		Price:       100,
		Destination: "Seoul",
	})
	if err != nil {
		log.Fatalf("Could not create orderId: %v", err)
	}
	log.Print("Order: ", orderId.String())

	retrievedOrder, err := orderClient.GetOrder(ctx, wrapperspb.String(orderId.GetValue()))
	if err != nil {
		log.Printf("Could not get orderId: %v", err)
	}
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
	if err != nil {
		log.Fatalf("%v.UpdateOrders(_) = _, %v", orderClient, err)
	}

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

	if err := updateStream.Send(updOrder1); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder1, err)
	}

	if err := updateStream.Send(updOrder2); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder2, err)
	}

	if err := updateStream.Send(updOrder3); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder3, err)
	}

	updateRes, err := updateStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", updateStream, err, nil)
	}
	log.Printf("Update Orders Res: %s", updateRes)

}
