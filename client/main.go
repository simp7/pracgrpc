package main

import (
	"context"
	"github.com/simp7/pracgrpc/model"
	pb "github.com/simp7/pracgrpc/model/ecommerce"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"log"
	"time"
)

const (
	address = "localhost:50051"
)

func orderUnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	log.Println("Method: " + method)
	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		log.Printf("Errors in %s: %v", method, err)
	}

	log.Println(reply)
	return err
}

func clientStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	log.Println("===== [Client Interceptor] ", method)
	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}
	return newWrappedStream(s), nil
}

type wrappedStream struct {
	grpc.ClientStream
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("===== [Client Stream Interceptor] "+"Receive a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf("===== [Client Stream Interceptor] "+"Send a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.SendMsg(m)
}

func newWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStream{s}
}

func createUser(userStore model.UserStore, username, password, role string) error {
	user, err := model.NewUser(username, password, role)
	if err != nil {
		return err
	}
	return userStore.Save(user)
}

func main() {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(orderUnaryClientInterceptor),
		grpc.WithStreamInterceptor(clientStreamInterceptor),
	}

	conn, err := grpc.Dial(address, opts...)
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
