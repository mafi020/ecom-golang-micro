package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		ENV            string `mapstructure:"ENV"`
		APIGatewayPort string `mapstructure:"API_GATEWAY_PORT"`

		CatalogServiceHTTPPort string `mapstructure:"CATALOG_SERVICE_HTTP_PORT"`
		CatalogServiceURL      string `mapstructure:"CATALOG_SERVICE_URL"`
		CatalogServiceGRPCPort string `mapstructure:"CATALOG_SERVICE_GRPC_PORT"`

		CartServiceHTTPPort string `mapstructure:"CART_SERVICE_HTTP_PORT"`
		CartServiceURL      string `mapstructure:"CART_SERVICE_URL"`
		CartServiceGRPCPort string `mapstructure:"CART_SERVICE_GRPC_PORT"`

		OrderServiceHTTPPort string `mapstructure:"ORDER_SERVICE_HTTP_PORT"`
		OrderServiceURL      string `mapstructure:"ORDER_SERVICE_URL"`
		OrderServiceGRPCPort string `mapstructure:"ORDER_SERVICE_GRPC_PORT"`

		PaymentServiceHTTPPort string `mapstructure:"PAYMENT_SERVICE_HTTP_PORT"`
		PaymentServiceURL      string `mapstructure:"PAYMENT_SERVICE_URL"`
		PaymentServiceGRPCPort string `mapstructure:"PAYMENT_SERVICE_GRPC_PORT"`

		IdentityServiceHTTPPort string `mapstructure:"IDENTITY_SERVICE_HTTP_PORT"`
		IdentityServiceURL      string `mapstructure:"IDENTITY_SERVICE_URL"`
		IdentityServiceGRPCPort string `mapstructure:"IDENTITY_SERVICE_GRPC_PORT"`
	} `mapstructure:",squash"` // Allows it to find PORT at the top level of .env

	Postgres struct {
		Host           string `mapstructure:"PG_HOST"`
		Port           int    `mapstructure:"PG_PORT"`
		User           string `mapstructure:"PG_USER"`
		Password       string `mapstructure:"PG_PASSWORD"`
		CatalogDBName  string `mapstructure:"PG_CATALOG_DB_NAME"`
		CartDBName     string `mapstructure:"PG_CART_DB_NAME"`
		OrderDBName    string `mapstructure:"PG_ORDER_DB_NAME"`
		PaymentDBName  string `mapstructure:"PG_PAYMENT_DB_NAME"`
		IdentityDBName string `mapstructure:"PG_IDENTITY_DB_NAME"`
	} `mapstructure:",squash"`

	JWT struct {
		Secret            string `mapstructure:"JWT_SECRET"`
		Expiration        string `mapstructure:"JWT_EXPIRATION"`
		RefreshExpiration string `mapstructure:"JWT_REFRESH_EXPIRATION"`
	} `mapstructure:",squash"`
}

func LoadConfig() *Config {
	// Point directly to your .env file
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	var cfg Config
	// Unmarshal uses the mapstructure tags
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Error unmarshalling config: %v", err)
	}

	return &cfg
}
