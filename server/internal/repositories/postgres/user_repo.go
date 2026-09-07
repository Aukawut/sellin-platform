package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sellin-server/internal/core/domain"
)

type UserRepo struct{ pool *pgxpool.Pool }

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

const userCols = `id, email, display_name, role, is_active, password_hash, created_at`

func scanUser(r pgx.Row) (domain.User, error) {
	var u domain.User
	err := r.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.IsActive, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

func (r *UserRepo) Create(ctx context.Context, u domain.User, hash string) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name, role, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userCols,
		domain.NormalizeEmail(u.Email), hash, u.DisplayName, string(u.Role), u.IsActive)

	created, err := scanUser(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.User{}, fmt.Errorf("อีเมล %s ถูกใช้ไปแล้ว", u.Email)
	}
	return created, err
}

func (r *UserRepo) ByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	return u, err
}

func (r *UserRepo) ByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE lower(email) = $1`, domain.NormalizeEmail(email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	return u, err
}

func (r *UserRepo) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+userCols+` FROM users ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepo) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET is_active = $2, updated_at = now() WHERE id = $1`, id, active)
	return err
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
	return err
}
