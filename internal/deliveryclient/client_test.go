package deliveryclient

import (
	"context"
	"testing"

	deliveryv1 "github.com/ThDawnWind/food-delivery-contracts/gen/go/delivery/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeDeliveryServiceClient struct {
	orderAmount   int64
	deliveryPrice int64
	err           error
}

func (f *fakeDeliveryServiceClient) CalculateDelivery(
	_ context.Context,
	req *deliveryv1.CalculateDeliveryRequest,
	_ ...grpc.CallOption,
) (*deliveryv1.CalculateDeliveryResponse, error) {
	f.orderAmount = req.GetOrderAmount()

	if f.err != nil {
		return nil, f.err
	}

	return &deliveryv1.CalculateDeliveryResponse{
		DeliveryPrice: f.deliveryPrice,
	}, nil
}

func TestClient_Calculate(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeDeliveryServiceClient{
		deliveryPrice: 19900,
	}

	client := &Client{
		grpcClient: fakeClient,
	}

	got, err := client.Calculate(context.Background(), 120000)
	if err != nil {
		t.Fatalf("Calculate() error = %v", err)
	}

	if fakeClient.orderAmount != 120000 {
		t.Fatalf(
			"order amount = %d, want %d",
			fakeClient.orderAmount,
			int64(120000),
		)
	}

	if got != 19900 {
		t.Fatalf("Calculate() = %d, want %d", got, int64(19900))
	}
}

func TestClient_Calculate_GRPCError(t *testing.T) {
	t.Parallel()

	fakeClient := &fakeDeliveryServiceClient{
		err: status.Error(
			codes.Unavailable,
			"delivery service unavailable",
		),
	}

	client := &Client{
		grpcClient: fakeClient,
	}

	got, err := client.Calculate(context.Background(), 120000)
	if err == nil {
		t.Fatal("Calculate() expected error, got nil")
	}

	if got != 0 {
		t.Fatalf("Calculate() = %d, want 0", got)
	}

	if status.Code(err) != codes.Unavailable {
		t.Fatalf(
			"error code = %s, want %s",
			status.Code(err),
			codes.Unavailable,
		)
	}
}
