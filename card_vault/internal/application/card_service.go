package application

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"huhnak/card_vault/internal/domain"
)

type CardRepository interface {
	Save(ctx context.Context, card *domain.TokenizedCard) error
	GetByToken(ctx context.Context, token string) (*domain.TokenizedCard, error)
}
type CardCache interface {
	SetToken(ctx context.Context, token string, card *domain.TokenizedCard, ttl time.Duration) error
	GetToken(ctx context.Context, token string) (*domain.TokenizedCard, error)
}

type CardService struct {
	repo  CardRepository
	cache CardCache
}

func NewCardService(repo CardRepository, cache CardCache) *CardService {
	return &CardService{
		repo:  repo,
		cache: cache,
	}
}

func (s *CardService) TokenizeCard(ctx context.Context, input domain.CardInput) (*domain.TokenizedCard, error) {
	cleanPan := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, input.CardNumber)
	if !ValidateLuhn(cleanPan) {
		return nil, domain.ErrInvalidCardNumber
	}
	now := time.Now()
	if input.ExpYear < now.Year() || (input.ExpYear == now.Year() && input.ExpMonth < int(now.Month())) {
		return nil, domain.ErrCardExpired
	}

	if len(input.CVV) < 3 || len(input.CVV) > 4 {
		return nil, domain.ErrInvalidCVV
	}
	token := generateSecureToken()
	card := &domain.TokenizedCard{
		ID:        token,
		MaskedPAN: MaskCardPan(cleanPan),
		ExpMonth:  input.ExpMonth,
		ExpYear:   input.ExpYear,
		CreatedAt: now.UTC(),
	}
	if err := s.repo.Save(ctx, card); err != nil {
		return nil, fmt.Errorf("failed to save card to db: %w", err)
	}
	if s.cache != nil {
		if err := s.cache.SetToken(ctx, token, card, 10*time.Minute); err != nil {
			slog.Error("failed to set card in cache", "error", err)
		}
	}
	return card, nil
}

func (s CardService) GetById(ctx context.Context, token string) (*domain.TokenizedCard, error) {
	if s.cache != nil {
		if cached, err := s.cache.GetToken(ctx, token); err == nil && cached != nil {
			return cached, nil
		}
	}
	card, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve card: %w", err)
	}
	if s.cache != nil {
		if err := s.cache.SetToken(ctx, token, card, 10*time.Minute); err != nil {
			slog.Error("failed to set card in cache", "token", token, "error", err)
		}
	}
	return card, nil
}

func ValidateLuhn(pan string) bool {
	if len(pan) < 13 || len(pan) > 19 {
		return false
	}
	var sum int
	alternate := false
	for i := len(pan) - 1; i >= 0; i-- {
		digit := int(pan[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alternate = !alternate
	}
	return sum%10 == 0
}

func generateSecureToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("token_%x-%x-%x-%x", b[0:4], b[4:8], b[8:12], b[12:16])
}

func MaskCardPan(pan string) string {
	if len(pan) < 10 {
		return "**********"
	}
	prefix := pan[:6]
	suffix := pan[len(pan)-4:]
	maskedLength := len(pan) - 10

	return prefix + strings.Repeat("*", maskedLength) + suffix
}
