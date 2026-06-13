package rpc_client

import (
	"fmt"
	"log"

	cartpb "github.com/mafi020/ecom-golang-micro/proto/cart"
	catalogpb "github.com/mafi020/ecom-golang-micro/proto/catalog"
	identitypb "github.com/mafi020/ecom-golang-micro/proto/identity"
	orderpb "github.com/mafi020/ecom-golang-micro/proto/order"
	paymentpb "github.com/mafi020/ecom-golang-micro/proto/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Clients holds all internal grpc service clients used across our Mono-repo binaries
type Clients struct {
	Catalog  catalogpb.CatalogServiceClient
	Cart     cartpb.CartServiceClient
	Order    orderpb.OrderServiceClient
	Payment  paymentpb.PaymentServiceClient
	Identity identitypb.IdentityServiceClient

	connections []*grpc.ClientConn
}

type Config struct {
	CatalogAddr  string
	CartAddr     string
	OrderAddr    string
	PaymentAddr  string
	IdentityAddr string
}

// InitializeClients sets up all required outbound grpc connection loops conditionally
func InitializeClients(cfg Config) (*Clients, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	clients := &Clients{
		connections: make([]*grpc.ClientConn, 0),
	}

	// 1. Conditionally Initialize Catalog Client
	if cfg.CatalogAddr != "" {
		log.Printf("[grpc Client Pool] Connecting to Catalog Service at %s...", cfg.CatalogAddr)
		catalogConn, err := grpc.NewClient(cfg.CatalogAddr, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to catalog service: %w", err)
		}
		clients.connections = append(clients.connections, catalogConn)
		clients.Catalog = catalogpb.NewCatalogServiceClient(catalogConn)
	}

	// 2. Conditionally Initialize Cart Client
	if cfg.CartAddr != "" {
		log.Printf("[grpc Client Pool] Connecting to Cart Service at %s...", cfg.CartAddr)
		cartConn, err := grpc.NewClient(cfg.CartAddr, dialOpts...)
		if err != nil {
			clients.Close() // Safely tear down open connections if this client fails
			return nil, fmt.Errorf("failed to connect to cart service: %w", err)
		}
		clients.connections = append(clients.connections, cartConn)
		clients.Cart = cartpb.NewCartServiceClient(cartConn)
	}

	// 3. Conditionally Initialize Order Client
	if cfg.OrderAddr != "" {
		log.Printf("[grpc Client Pool] Connecting to Order Service at %s...", cfg.OrderAddr)
		orderConn, err := grpc.NewClient(cfg.OrderAddr, dialOpts...)
		if err != nil {
			clients.Close() // Safely tear down open connections if this client fails
			return nil, fmt.Errorf("failed to connect to order service: %w", err)
		}
		clients.connections = append(clients.connections, orderConn)
		clients.Order = orderpb.NewOrderServiceClient(orderConn)
	}

	// 4. Conditionally Initialize Payment Client
	if cfg.PaymentAddr != "" {
		log.Printf("[grpc Client Pool] Connecting to Payment Service at %s...", cfg.PaymentAddr)
		paymentConn, err := grpc.NewClient(cfg.PaymentAddr, dialOpts...)
		if err != nil {
			clients.Close() // Safely tear down open connections if this client fails
			return nil, fmt.Errorf("failed to connect to payment service: %w", err)
		}
		clients.connections = append(clients.connections, paymentConn)
		clients.Payment = paymentpb.NewPaymentServiceClient(paymentConn)
	}

	// 5. Conditionally Initialize Identiy Client
	if cfg.IdentityAddr != "" {
		log.Printf("[grpc Client Pool] Connecting to Identity Service at %s...", cfg.IdentityAddr)
		identityConn, err := grpc.NewClient(cfg.IdentityAddr, dialOpts...)
		if err != nil {
			clients.Close() // Safely tear down open connections if this client fails
			return nil, fmt.Errorf("failed to connect to identity service: %w", err)
		}
		clients.connections = append(clients.connections, identityConn)
		clients.Identity = identitypb.NewIdentityServiceClient(identityConn)
	}

	return clients, nil
}

// Close gracefully terminates all active grpc client connection pools
func (c *Clients) Close() {
	log.Println("[grpc Client Pool] Shutting down active microservice connections...")
	for _, conn := range c.connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				log.Printf("Warning: error closing grpc connection: %v", err)
			}
		}
	}
}
