package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresContainer wraps a PostgreSQL test container.
type PostgresContainer struct {
	Container testcontainers.Container
	DB        *gorm.DB
	ConnStr   string
}

// SetupPostgres creates a PostgreSQL test container.
func SetupPostgres(ctx context.Context) (*PostgresContainer, error) {
	container, err := tcPostgres.Run(ctx,
		"postgres:15-alpine",
		tcPostgres.WithDatabase("testdb"),
		tcPostgres.WithUsername("test"),
		tcPostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	db, err := gorm.Open(gormPostgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return &PostgresContainer{
		Container: container,
		DB:        db,
		ConnStr:   connStr,
	}, nil
}

// Cleanup terminates the container.
func (p *PostgresContainer) Cleanup(ctx context.Context) error {
	if p.Container != nil {
		return p.Container.Terminate(ctx)
	}
	return nil
}

// MongoContainer wraps a MongoDB test container.
type MongoContainer struct {
	Container testcontainers.Container
	Client    *mongo.Client
	Database  *mongo.Database
	ConnStr   string
}

// SetupMongo creates a MongoDB test container.
func SetupMongo(ctx context.Context) (*MongoContainer, error) {
	container, err := mongodb.Run(ctx,
		"mongo:7",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Waiting for connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start mongodb container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(connStr))
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return &MongoContainer{
		Container: container,
		Client:    client,
		Database:  client.Database("testdb"),
		ConnStr:   connStr,
	}, nil
}

// Cleanup terminates the container.
func (m *MongoContainer) Cleanup(ctx context.Context) error {
	if m.Client != nil {
		m.Client.Disconnect(ctx)
	}
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}
	return nil
}

// RedisContainer wraps a Redis test container.
type RedisContainer struct {
	Container testcontainers.Container
	Client    *redis.Client
	ConnStr   string
}

// SetupRedis creates a Redis test container.
func SetupRedis(ctx context.Context) (*RedisContainer, error) {
	container, err := tcRedis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start redis container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	opt, err := redis.ParseURL(connStr)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisContainer{
		Container: container,
		Client:    client,
		ConnStr:   connStr,
	}, nil
}

// Cleanup terminates the container.
func (r *RedisContainer) Cleanup(ctx context.Context) error {
	if r.Client != nil {
		r.Client.Close()
	}
	if r.Container != nil {
		return r.Container.Terminate(ctx)
	}
	return nil
}
