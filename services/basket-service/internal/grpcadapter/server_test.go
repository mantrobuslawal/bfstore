package grpcadapter

import (
	"io"
	"log/slog"
	"testing"

	"github.com/mantrobuslawal/bfstore/services/basket-service/internal/basket"
	"google.golang.org/grpc"
)

func TestNewServerRegistersBasketService(t *testing.T) {
	t.Parallel()

	service := new(basket.Service)

	server, err := NewServer(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	defer server.Stop()

	serviceInfo := server.GetServiceInfo()

	basketServiceInfo, ok := serviceInfo["bfstore.basket.v1.BasketService"]
	if !ok {
		t.Fatalf("registered services = %#v, want bfstore.basket.v1.BasketService", serviceInfo)
	}

	assertRegisteredMethod(t, basketServiceInfo.Methods, "CreateBasket")
	assertRegisteredMethod(t, basketServiceInfo.Methods, "GetBasket")
	assertRegisteredMethod(t, basketServiceInfo.Methods, "AddItem")
	assertRegisteredMethod(t, basketServiceInfo.Methods, "UpdateItemQuantity")
	assertRegisteredMethod(t, basketServiceInfo.Methods, "RemoveItem")
	assertRegisteredMethod(t, basketServiceInfo.Methods, "ClearBasket")
}

func TestNewServerPanicsWhenBasketServiceIsNil(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("NewServer(nil, logger) did not panic, want panic")
		}
	}()

	_, _ = NewServer(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func assertRegisteredMethod(t *testing.T, methods []grpc.MethodInfo, wantName string) {
	t.Helper()

	for _, method := range methods {
		if method.Name == wantName {
			return
		}
	}

	t.Fatalf("registered methods = %#v, want method %q", methods, wantName)
}
