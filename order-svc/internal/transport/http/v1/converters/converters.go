package converters

import (
	"fmt"

	"github.com/corray333/backend-labs/order/internal/service/models/currency"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/corray333/backend-labs/order/internal/service/models/orderitem"
	pb "github.com/corray333/backend-labs/order/pkg/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OrderItemToProto converts internal OrderItem model to protobuf OrderItem.
func OrderItemToProto(item orderitem.OrderItem) *pb.OrderItem {
	return &pb.OrderItem{
		ProductId:     item.ProductID,
		Quantity:      int32(item.Quantity),
		ProductTitle:  item.ProductTitle,
		ProductUrl:    item.ProductUrl,
		PriceCents:    item.PriceCents,
		PriceCurrency: item.PriceCurrency.String(),
	}
}

// OrderItemFromProto converts protobuf OrderItem to internal OrderItem model.
func OrderItemFromProto(pbItem *pb.OrderItem) (*orderitem.OrderItem, error) {
	cur, err := currency.ParseCurrency(pbItem.PriceCurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to parse currency: %w", err)
	}

	return &orderitem.OrderItem{
		ProductID:     pbItem.ProductId,
		Quantity:      int(pbItem.Quantity),
		ProductTitle:  pbItem.ProductTitle,
		ProductUrl:    pbItem.ProductUrl,
		PriceCents:    pbItem.PriceCents,
		PriceCurrency: cur,
		// ID, OrderID, CreatedAt, UpdatedAt will be set by the service layer
	}, nil
}

// OrderToProto converts internal Order model to protobuf Order.
func OrderToProto(o order.Order) *pb.Order {
	items := make([]*pb.OrderItem, len(o.OrderItems))
	for i, item := range o.OrderItems {
		items[i] = OrderItemToProto(item)
	}

	return &pb.Order{
		Id:                 o.ID,
		CustomerId:         o.CustomerID,
		DeliveryAddress:    o.DeliveryAddress,
		TotalPriceCents:    o.TotalPriceCents,
		TotalPriceCurrency: o.TotalPriceCurrency.String(),
		CreatedAt:          timestamppb.New(o.CreatedAt),
		UpdatedAt:          timestamppb.New(o.UpdatedAt),
		OrderItems:         items,
	}
}

// OrderFromProto converts protobuf Order to internal Order model.
func OrderFromProto(pbOrder *pb.Order) (*order.Order, error) {
	cur, err := currency.ParseCurrency(pbOrder.TotalPriceCurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to parse currency: %w", err)
	}

	items := make([]orderitem.OrderItem, len(pbOrder.OrderItems))
	for i, pbItem := range pbOrder.OrderItems {
		item, err := OrderItemFromProto(pbItem)
		if err != nil {
			return nil, fmt.Errorf("failed to convert order item %d: %w", i, err)
		}
		items[i] = *item
	}

	o := &order.Order{
		CustomerID:         pbOrder.CustomerId,
		DeliveryAddress:    pbOrder.DeliveryAddress,
		TotalPriceCents:    pbOrder.TotalPriceCents,
		TotalPriceCurrency: cur,
		OrderItems:         items,
	}

	// Set timestamps if they exist in protobuf
	if pbOrder.CreatedAt != nil {
		o.CreatedAt = pbOrder.CreatedAt.AsTime()
	}
	if pbOrder.UpdatedAt != nil {
		o.UpdatedAt = pbOrder.UpdatedAt.AsTime()
	}
	if pbOrder.Id != 0 {
		o.ID = pbOrder.Id
	}

	return o, nil
}

// BatchInsertRequestFromProto converts protobuf BatchInsertRequest to slice of internal Order models.
func BatchInsertRequestFromProto(req *pb.BatchInsertRequest) ([]order.Order, error) {
	orders := make([]order.Order, len(req.Orders))
	for i, pbOrder := range req.Orders {
		o, err := OrderFromProto(pbOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to convert order %d: %w", i, err)
		}
		orders[i] = *o
	}

	return orders, nil
}

// BatchInsertResponseToProto converts slice of internal Order models to protobuf BatchInsertResponse.
func BatchInsertResponseToProto(orders []order.Order) *pb.BatchInsertResponse {
	pbOrders := make([]*pb.Order, len(orders))
	for i, o := range orders {
		pbOrders[i] = OrderToProto(o)
	}

	return &pb.BatchInsertResponse{
		Orders: pbOrders,
	}
}

// ListOrdersRequestFromProto converts protobuf ListOrdersRequest to internal QueryOrderItemsModel.
func ListOrdersRequestFromProto(req *pb.ListOrdersRequest) orderitem.QueryOrderItemsModel {
	return orderitem.QueryOrderItemsModel{
		Ids:         req.Ids,
		CustomerIds: req.CustomerIds,
		Limit:       int(req.Limit),
		Offset:      int(req.Offset),
	}
}

// ListOrdersResponseToProto converts slice of internal Order models to protobuf ListOrdersResponse.
func ListOrdersResponseToProto(orders []order.Order) *pb.ListOrdersResponse {
	pbOrders := make([]*pb.Order, len(orders))
	for i, o := range orders {
		pbOrders[i] = OrderToProto(o)
	}

	return &pb.ListOrdersResponse{
		Orders: pbOrders,
	}
}
