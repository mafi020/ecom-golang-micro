package config

import "fmt"

func (c *Config) PostgresDSN(user, password, host, dbname string, port int) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		dbname,
	)
}
