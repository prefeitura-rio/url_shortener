package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"url_shortener/internal/database"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
	ttl    time.Duration
}

func Init(redisURL string, ttl time.Duration) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &Client{
		client: client,
		ttl:    ttl,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Client) GetURL(ctx context.Context, shortPath string) (*database.URL, error) {
	key := fmt.Sprintf("url:%s", shortPath)
	
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	var url database.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, fmt.Errorf("failed to unmarshal URL: %w", err)
	}

	return &url, nil
}

func (c *Client) SetURL(ctx context.Context, shortPath string, url *database.URL) error {
	key := fmt.Sprintf("url:%s", shortPath)
	
	data, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("failed to marshal URL: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set in Redis: %w", err)
	}

	return nil
}

func (c *Client) DeleteURL(ctx context.Context, shortPath string) error {
	key := fmt.Sprintf("url:%s", shortPath)
	
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	return nil
}

func (c *Client) GetURLByID(ctx context.Context, id string) (*database.URL, error) {
	key := fmt.Sprintf("url_id:%s", id)
	
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	var url database.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, fmt.Errorf("failed to unmarshal URL: %w", err)
	}

	return &url, nil
}

func (c *Client) SetURLByID(ctx context.Context, id string, url *database.URL) error {
	key := fmt.Sprintf("url_id:%s", id)
	
	data, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("failed to marshal URL: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set in Redis: %w", err)
	}

	return nil
}

func (c *Client) DeleteURLByID(ctx context.Context, id string) error {
	key := fmt.Sprintf("url_id:%s", id)
	
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	return nil
} 