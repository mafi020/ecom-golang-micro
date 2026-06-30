package config

import (
	"log"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		ENV            string `mapstructure:"ENV"`
		RabbitMqURL    string `mapstructure:"RABBITMQ_URL"`
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
		PgSharedUser     string `mapstructure:"PG_SHARED_USER"`
		PgSharedPassword string `mapstructure:"PG_SHARED_PASSWORD"`

		PgIdentityHost     string `mapstructure:"PG_IDENTITY_HOST"`
		PgIdentityPort     int    `mapstructure:"PG_IDENTITY_PORT"`
		PgIdentityUser     string `mapstructure:"PG_IDENTITY_USER"`
		PgIdentityPassword string `mapstructure:"PG_IDENTITY_PASSWORD"`
		PgIdentityDBName   string `mapstructure:"PG_IDENTITY_DB_NAME"`

		PgCatalogHost     string `mapstructure:"PG_CATALOG_HOST"`
		PgCatalogPort     int    `mapstructure:"PG_CATALOG_PORT"`
		PgCatalogUser     string `mapstructure:"PG_CATALOG_USER"`
		PgCatalogPassword string `mapstructure:"PG_CATALOG_PASSWORD"`
		PgCatalogDBName   string `mapstructure:"PG_CATALOG_DB_NAME"`

		PgCartHost     string `mapstructure:"PG_CART_HOST"`
		PgCartPort     int    `mapstructure:"PG_CART_PORT"`
		PgCartUser     string `mapstructure:"PG_CART_USER"`
		PgCartPassword string `mapstructure:"PG_CART_PASSWORD"`
		PgCartDBName   string `mapstructure:"PG_CART_DB_NAME"`

		PgOrderHost     string `mapstructure:"PG_ORDER_HOST"`
		PgOrderPort     int    `mapstructure:"PG_ORDER_PORT"`
		PgOrderUser     string `mapstructure:"PG_ORDER_USER"`
		PgOrderPassword string `mapstructure:"PG_ORDER_PASSWORD"`
		PgOrderDBName   string `mapstructure:"PG_ORDER_DB_NAME"`

		PgPaymentHost     string `mapstructure:"PG_PAYMENT_HOST"`
		PgPaymentPort     int    `mapstructure:"PG_PAYMENT_PORT"`
		PgPaymentUser     string `mapstructure:"PG_PAYMENT_USER"`
		PgPaymentPassword string `mapstructure:"PG_PAYMENT_PASSWORD"`
		PgPaymentDBName   string `mapstructure:"PG_PAYMENT_DB_NAME"`
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
		log.Println("No physical .env file found. Reading directly from system environment.")
	}

	viper.AutomaticEnv()

	var cfg Config
	bindEnvKeys(reflect.TypeOf(cfg))

	// Unmarshal uses the mapstructure tags
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Error unmarshalling config: %v", err)
	}

	return &cfg
}

func bindEnvKeys(t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// If it's a nested struct with a squash tag, recursively look inside it
		if field.Type.Kind() == reflect.Struct {
			tag := field.Tag.Get("mapstructure")
			if strings.Contains(tag, ",squash") {
				bindEnvKeys(field.Type)
			}
			continue
		}

		// Get the mapstructure key name
		key := field.Tag.Get("mapstructure")
		if key != "" && key != ",squash" {
			viper.BindEnv(key)
		}
	}
}
