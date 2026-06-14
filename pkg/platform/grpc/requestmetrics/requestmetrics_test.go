package requestmetrics

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
)

func TestUnaryServerInterceptorRequiresServiceName(t *testing.T) {
	t.Parallel()

	interceptor, err := UnaryServerInterceptor(Config{
		ServiceName: " ",
	})

	if err == nil {
		t.Fatal("UnaryServerInterceptor() error = nil, want non-nil")
	}

	if interceptor != nil {
		t.Fatal("UnaryServerInterceptor() interceptor != nil, want nil")
	}
}

func TestUnaryServerInterceptorReturnsInterceptor(t *testing.T) {
	t.Parallel()

	interceptor, err := UnaryServerInterceptor(Config{
		ServiceName: "catalog-service",
	})

	if err != nil {
		t.Fatalf("UnaryServerInterceptor() error = %v, want nil", err)
	}

	if interceptor == nil {
		t.Fatal("UnaryServerInterceptor() interceptor = nil, want non-nil")
	}
}

func TestUnaryServerInterceptorCallsHandlerAndReturnsResponse(t *testing.T) {
	t.Parallel()

	interceptor, err := UnaryServerInterceptor(Config{
		ServiceName: "catalog-service",
	})
	if err != nil {
		t.Fatalf("UnaryServerInterceptor() error = %v, want nil", err)
	}

	wantResp := "ok"

	resp, err := interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{
			FullMethod: "/bfstore.catalog.v1.CatalogService/ListProducts",
		},
		func(ctx context.Context, req any) (any, error) {
			return wantResp, nil
		},
	)

	if err != nil {
		t.Fatalf("interceptor() error = %v, want nil", err)
	}

	if resp != wantResp {
		t.Fatalf("interceptor() response = %v, want %v", resp, wantResp)
	}
}

func TestUnaryServerInterceptorReturnsHandlerError(t *testing.T) {
	t.Parallel()

	interceptor, err := UnaryServerInterceptor(Config{
		ServiceName: "catalog-service",
	})
	if err != nil {
		t.Fatalf("UnaryServerInterceptor() error = %v, want nil", err)
	}

	wantErr := errors.New("handler failed")

	_, err = interceptor(
		context.Background(),
		nil,
		&grpc.UnaryServerInfo{
			FullMethod: "/bfstore.catalog.v1.CatalogService/ListProducts",
		},
		func(ctx context.Context, req any) (any, error) {
			return nil, wantErr
		},
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("interceptor() error = %v, want %v", err, wantErr)
	}
}

func TestSplitFullMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fullMethod  string
		wantService string
		wantMethod  string
	}{
		{
			name:        "valid full method",
			fullMethod:  "/bfstore.catalog.v1.CatalogService/ListProducts",
			wantService: "bfstore.catalog.v1.CatalogService",
			wantMethod:  "ListProducts",
		},
		{
			name:        "missing leading slash",
			fullMethod:  "bfstore.catalog.v1.CatalogService/ListProducts",
			wantService: "bfstore.catalog.v1.CatalogService",
			wantMethod:  "ListProducts",
		},
		{
			name:        "empty",
			fullMethod:  "",
			wantService: "unknown",
			wantMethod:  "unknown",
		},
		{
			name:        "unexpected shape",
			fullMethod:  "/unexpected",
			wantService: "unexpected",
			wantMethod:  "unknown",
		},
		{
			name:        "empty method",
			fullMethod:  "/bfstore.catalog.v1.CatalogService/",
			wantService: "bfstore.catalog.v1.CatalogService",
			wantMethod:  "unknown",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotService, gotMethod := splitFullMethod(tt.fullMethod)

			if gotService != tt.wantService {
				t.Fatalf("service = %q, want %q", gotService, tt.wantService)
			}

			if gotMethod != tt.wantMethod {
				t.Fatalf("method = %q, want %q", gotMethod, tt.wantMethod)
			}
		})
	}
}
