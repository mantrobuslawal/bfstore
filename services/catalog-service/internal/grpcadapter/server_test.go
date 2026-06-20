package grpcadapter

import (
	"io"
	"log/slog"
	"testing"

	"github.com/mantrobuslawal/bfstore/services/catalog-service/internal/catalog"
	"google.golang.org/grpc"
)

func TestNewServerRegistersCatalogService(t *testing.T) {
	t.Parallel()

	server, err := NewServer(new(catalog.Service), servertestLogger())
	if err != nil {
		t.Fatalf("NewServer() error = %v, want nil", err)
	}
	defer server.Stop()

	serviceInfo := server.GetServiceInfo()

	catalogServiceInfo, ok := serviceInfo["bfstore.catalog.v1.CatalogService"]
	if !ok {
		t.Fatalf("registered services = %#v, want bfstore.catalog.v1.CatalogService", serviceInfo)
	}

	assertRegisteredMethod(t, catalogServiceInfo.Methods, "ListProducts")
	assertRegisteredMethod(t, catalogServiceInfo.Methods, "GetProduct")
	assertRegisteredMethod(t, catalogServiceInfo.Methods, "ValidateProductVariant")
	assertRegisteredMethod(t, catalogServiceInfo.Methods, "ListCategories")
	assertRegisteredMethod(t, catalogServiceInfo.Methods, "ListProductAttributeDefinitions")
}

func TestNewServerPanicsWhenCatalogServiceIsNil(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("NewServer(nil, logger) did not panic, want panic")
		}
	}()

	_, _ = NewServer(nil, testLogger())
}

func assertRegisteredMethod(
	t *testing.T,
	methods []grpc.MethodInfo,
	wantName string,
) {
	t.Helper()

	for _, method := range methods {
		if method.Name == wantName {
			return
		}
	}

	t.Fatalf("registered methods = %#v, want method %q", methods, wantName)
}

func servertestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
