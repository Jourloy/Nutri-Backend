package cache

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/redis/go-redis/v9"

	"github.com/jourloy/nutri02/internal/lib"
)

var (
	client *redis.Client

	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[rdis]",
		Level:  log.DebugLevel,
	})
)

// Client returns initialized redis client instance.
func Client() *redis.Client {
	return client
}

// Connect initializes redis client using environment configuration.
func Connect() error {
	if client != nil {
		return nil
	}

	addr := fmt.Sprintf("%s:%s", lib.Config.RedisHost, lib.Config.RedisPort)
	client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: lib.Config.RedisPassword,
		DB:       lib.Config.RedisDB,
		Protocol: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect redis", "error", err)
		client = nil
		return err
	}

	logger.Info("connected to redis", "addr", addr)
	return nil
}

// Close closes redis client if initialized.
func Close() error {
	if client == nil {
		return nil
	}
	if err := client.Close(); err != nil {
		logger.Error("failed to close redis", "error", err)
		return err
	}
	client = nil
	return nil
}
