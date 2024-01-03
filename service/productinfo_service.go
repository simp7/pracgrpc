package main

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	pb "github.com/simp7/pracgrpc/model/ecommerce"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"log"
	"strings"
)

type server struct {
	productMap          map[string]*pb.Product
	orderMap            map[string]*pb.Order
	combinedShipmentMap map[string]*pb.CombinedShipment
	batchSize           int
	pb.UnimplementedProductInfoServer
	pb.UnimplementedOrderManagementServer
}

func (s *server) AddProduct(ctx context.Context, in *pb.Product) (*pb.ProductID, error) {
	out, err := uuid.NewV4()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error while generating Product ID: %v", err)
	}
	in.Id = out.String()
	if s.productMap == nil {
		s.productMap = make(map[string]*pb.Product)
	}

	s.productMap[in.Id] = in
	return &pb.ProductID{Value: in.Id}, status.New(codes.OK, "").Err()
}

func (s *server) GetProduct(ctx context.Context, in *pb.ProductID) (*pb.Product, error) {
	value, exists := s.productMap[in.Value]
	if exists {
		return value, status.New(codes.OK, "").Err()
	}
	return nil, status.Errorf(codes.NotFound, "Product does not exist")
}

func (s *server) GetOrder(ctx context.Context, orderId *wrapperspb.StringValue) (*pb.Order, error) {
	log.Print("value", orderId.Value)
	ord, exists := s.orderMap[orderId.Value]
	if exists {
		return ord, status.New(codes.OK, "").Err()
	}
	return nil, status.Errorf(codes.NotFound, "Order does not exist")
}

func (s *server) SearchOrders(searchQuery *wrapperspb.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {
	for key, order := range s.orderMap {
		log.Print(key, order)
		for _, itemStr := range order.Items {
			log.Print(itemStr)
			if strings.Contains(itemStr, searchQuery.Value) {
				err := stream.Send(order)
				if err != nil {
					return fmt.Errorf("error sending message to stream: %v", err)
				}
				log.Print("Matching Order Found: ", key)
				break
			}
		}
	}
	return nil
}

func (s *server) CreateOrder(ctx context.Context, order *pb.Order) (*wrapperspb.StringValue, error) {
	out, err := uuid.NewV4()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error while generating Product ID: %v", err)
	}
	order.Id = out.String()

	if s.orderMap == nil {
		s.orderMap = make(map[string]*pb.Order)
	}

	s.orderMap[order.Id] = order
	return wrapperspb.String(order.Id), nil
}

func (s *server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {
	ordersStr := "UpdatedOrderIDs: "
	for {
		order, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(wrapperspb.String("Orders processed " + ordersStr))
		}
		s.orderMap[order.Id] = order
		log.Printf("Order ID ", order.Id, ": Updated")
		ordersStr += order.Id + ", "
	}
}

func (s *server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {
	s.combinedShipmentMap = make(map[string]*pb.CombinedShipment)
	batchMarker := 0
	for {
		orderId, err := stream.Recv()
		s.combinedShipmentMap[orderId.Value] = &pb.CombinedShipment{
			Id:         "...",
			Status:     "..",
			OrdersList: []*pb.Order{},
		}
		if err == io.EOF {
			for _, comb := range s.combinedShipmentMap {
				stream.Send(comb)
			}
			return nil
		}
		if err != nil {
			return err
		}
		if batchMarker == s.batchSize {
			for _, comb := range s.combinedShipmentMap {
				stream.Send(comb)
			}
			batchMarker = 0
			s.combinedShipmentMap = make(map[string]*pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}
