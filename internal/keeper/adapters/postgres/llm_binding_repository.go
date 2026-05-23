package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
)

// LLMBindingRepository implements the outbound.LLMBindingRepository port interface using PostgreSQL.
type LLMBindingRepository struct {
	pool *pgxpool.Pool
}

// NewLLMBindingRepository creates a new instance of LLMBindingRepository.
func NewLLMBindingRepository(pool *pgxpool.Pool) *LLMBindingRepository {
	return &LLMBindingRepository{pool: pool}
}

// Create inserts a new LLM provider binding into the database.
func (r *LLMBindingRepository) Create(ctx context.Context, b *domain.LLMBinding) error {
	query := `INSERT INTO llm_bindings (
		id, name, description, provider, api_base_url, api_key_secret_ref, 
		default_model, default_temperature, default_max_tokens, timeout_seconds, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.pool.Exec(ctx, query,
		b.ID, b.Name, b.Description, b.Provider, b.APIBaseURL, b.APIKeySecretRef,
		b.DefaultModel, b.DefaultTemperature, b.DefaultMaxTokens, b.TimeoutSeconds, b.Status, b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create llm binding: %w", err)
	}
	return nil
}

// GetByID retrieves an LLM provider binding by its ID.
func (r *LLMBindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LLMBinding, error) {
	query := `SELECT 
		id, name, description, provider, api_base_url, api_key_secret_ref, 
		default_model, default_temperature, default_max_tokens, timeout_seconds, status, created_at, updated_at
	FROM llm_bindings WHERE id = $1`

	var b domain.LLMBinding
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.Name, &b.Description, &b.Provider, &b.APIBaseURL, &b.APIKeySecretRef,
		&b.DefaultModel, &b.DefaultTemperature, &b.DefaultMaxTokens, &b.TimeoutSeconds, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("llm binding not found: %s", id)
		}
		return nil, fmt.Errorf("get llm binding by id: %w", err)
	}
	return &b, nil
}

// GetByName retrieves an LLM provider binding by its unique name.
func (r *LLMBindingRepository) GetByName(ctx context.Context, name string) (*domain.LLMBinding, error) {
	query := `SELECT 
		id, name, description, provider, api_base_url, api_key_secret_ref, 
		default_model, default_temperature, default_max_tokens, timeout_seconds, status, created_at, updated_at
	FROM llm_bindings WHERE name = $1`

	var b domain.LLMBinding
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&b.ID, &b.Name, &b.Description, &b.Provider, &b.APIBaseURL, &b.APIKeySecretRef,
		&b.DefaultModel, &b.DefaultTemperature, &b.DefaultMaxTokens, &b.TimeoutSeconds, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("llm binding not found by name: %s", name)
		}
		return nil, fmt.Errorf("get llm binding by name: %w", err)
	}
	return &b, nil
}

// List retrieves all LLM provider bindings.
func (r *LLMBindingRepository) List(ctx context.Context) ([]*domain.LLMBinding, error) {
	query := `SELECT 
		id, name, description, provider, api_base_url, api_key_secret_ref, 
		default_model, default_temperature, default_max_tokens, timeout_seconds, status, created_at, updated_at
	FROM llm_bindings ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list llm bindings: %w", err)
	}
	defer rows.Close()

	var bindings []*domain.LLMBinding
	for rows.Next() {
		var b domain.LLMBinding
		err := rows.Scan(
			&b.ID, &b.Name, &b.Description, &b.Provider, &b.APIBaseURL, &b.APIKeySecretRef,
			&b.DefaultModel, &b.DefaultTemperature, &b.DefaultMaxTokens, &b.TimeoutSeconds, &b.Status, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan llm binding: %w", err)
		}
		bindings = append(bindings, &b)
	}
	return bindings, nil
}

// Update updates an existing LLM provider binding.
func (r *LLMBindingRepository) Update(ctx context.Context, b *domain.LLMBinding) error {
	query := `UPDATE llm_bindings SET 
		name = $1, description = $2, provider = $3, api_base_url = $4, api_key_secret_ref = $5, 
		default_model = $6, default_temperature = $7, default_max_tokens = $8, timeout_seconds = $9, 
		status = $10, updated_at = $11
	WHERE id = $12`

	b.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query,
		b.Name, b.Description, b.Provider, b.APIBaseURL, b.APIKeySecretRef,
		b.DefaultModel, b.DefaultTemperature, b.DefaultMaxTokens, b.TimeoutSeconds, b.Status, b.UpdatedAt, b.ID,
	)
	if err != nil {
		return fmt.Errorf("update llm binding: %w", err)
	}
	return nil
}

// Delete removes an LLM provider binding from the database by its ID.
func (r *LLMBindingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM llm_bindings WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete llm binding: %w", err)
	}
	return nil
}
