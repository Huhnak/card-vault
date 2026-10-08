package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"huhnak/card_vault/internal/domain"
)

type CardRepository struct {
	db *sql.DB
}

func NewCardRepository(db *sql.DB) CardRepository {
	return CardRepository{
		db: db,
	}
}
func (r *CardRepository) Save(ctx context.Context, card *domain.TokenizedCard) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	queryCard := `
		INSERT INTO cards (id, masked_pan, exp_month, exp_year, created_at)
		VALUES ($1, $2, $3, $4, $5);
	`
	_, err = tx.ExecContext(ctx, queryCard,
		card.ID,
		card.MaskedPAN,
		card.ExpMonth,
		card.ExpYear,
		card.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert card: %w", err)
	}

	queryAudit := `
		INSERT INTO token_audit_logs (id, action, created_at)
		VALUES ($1, $2, $3)
	`
	_, err = tx.ExecContext(ctx, queryAudit,
		card.ID,
		"TOKEN_CREATED",
		card.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *CardRepository) GetByToken(ctx context.Context, token string) (*domain.TokenizedCard, error) {
	query := `
		SELECT id, masked_pan, exp_month, exp_year, created_at
		FROM cards
		WHERE id = $1
	`
	var card domain.TokenizedCard
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&card.ID,
		&card.MaskedPAN,
		&card.ExpMonth,
		&card.ExpYear,
		&card.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to query card: %w", err)
	}
	return &card, nil
}
