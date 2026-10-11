package deliveryclient

import (
	"context"

	deliveryv1 "github.com/ThDawnWind/food-delivery-contracts/gen/go/delivery/v1"
	"google.golang.org/grpc"
)

type Client struct {
	grpcClient deliveryv1.DeliveryServiceClient
}

func New(conn grpc.ClientConnInterface) *Client {
	return &Client{
		grpcClient: deliveryv1.NewDeliveryServiceClient(conn),
	}
}

func (c *Client) Calculate(
	ctx context.Context,
	orderAmount int64,
) (int64, error) {
	resp, err := c.grpcClient.CalculateDelivery(
		ctx,
		&deliveryv1.CalculateDeliveryRequest{
			OrderAmount: orderAmount,
		},
	)
	if err != nil {
		return 0, err
	}

	return resp.GetDeliveryPrice(), nil
}
