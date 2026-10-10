package rediscache

import (
	"context"
	"encoding/json"
	"fmt"
	"huhnak/card_vault/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type CardCache struct {
	client *redis.Client
}

func NewCardCache(client *redis.Client) *CardCache {
	return &CardCache{
		client: client,
	}
}

func (c *CardCache) SetToken(ctx context.Context, token string, card *domain.TokenizedCard, ttl time.Duration) error {
	data, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	err = c.client.Set(ctx, getKeyByToken(token), data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set card: %w", err)
	}
	return nil
}

func (c *CardCache) GetToken(ctx context.Context, token string) (*domain.TokenizedCard, error) {
	data, err := c.client.Get(ctx, getKeyByToken(token)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}

	var card domain.TokenizedCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal card: %w", err)
	}

	return &card, nil
}

func getKeyByToken(token string) string {
	return "card:" + token
}
