// Package auth implements password hashing and bearer tokens that are
// byte-for-byte compatible with Laravel Sanctum's personal access tokens, so
// tokens issued by the previous backend keep working after the switch.
package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/asihat/b2b-marketplace/backend/internal/db"
	"github.com/asihat/b2b-marketplace/backend/internal/strx"
)

// TokenableType is what Sanctum wrote into personal_access_tokens; kept so
// pre-existing rows keep resolving.
const TokenableType = "App\\Models\\User"

var ErrInvalidToken = errors.New("invalid token")

func HashPassword(plain string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	return string(h), err
}

// CheckPassword accepts $2y$ hashes produced by PHP as well as Go's $2a$.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// IssueToken creates a personal access token for a user and returns the
// plain text form "{id}|{secret}" exactly like Sanctum's plainTextToken.
func IssueToken(ctx context.Context, q db.Querier, userID int64, name string) (string, error) {
	plain := strx.Random(40)
	now := time.Now().UTC()

	var id int64
	err := q.QueryRow(ctx, `
		INSERT INTO personal_access_tokens (tokenable_type, tokenable_id, name, token, abilities, created_at, updated_at)
		VALUES ($1, $2, $3, $4, '["*"]', $5, $5)
		RETURNING id`,
		TokenableType, userID, name, hashToken(plain), now,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10) + "|" + plain, nil
}

// ResolveToken validates a bearer token and returns the owning user id and
// token id. Expired or unknown tokens yield ErrInvalidToken.
func ResolveToken(ctx context.Context, q db.Querier, bearer string) (userID, tokenID int64, err error) {
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return 0, 0, ErrInvalidToken
	}

	var (
		storedHash string
		expiresAt  *time.Time
	)

	if id, plain, ok := strings.Cut(bearer, "|"); ok {
		tokenID, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return 0, 0, ErrInvalidToken
		}
		err = q.QueryRow(ctx, `SELECT tokenable_id, token, expires_at FROM personal_access_tokens WHERE id = $1 AND tokenable_type = $2`,
			tokenID, TokenableType).Scan(&userID, &storedHash, &expiresAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, 0, ErrInvalidToken
			}
			return 0, 0, err
		}
		if subtle.ConstantTimeCompare([]byte(storedHash), []byte(hashToken(plain))) != 1 {
			return 0, 0, ErrInvalidToken
		}
	} else {
		// Legacy form without the id prefix: look the hash up directly.
		err = q.QueryRow(ctx, `SELECT id, tokenable_id, token, expires_at FROM personal_access_tokens WHERE token = $1 AND tokenable_type = $2`,
			hashToken(bearer), TokenableType).Scan(&tokenID, &userID, &storedHash, &expiresAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, 0, ErrInvalidToken
			}
			return 0, 0, err
		}
	}

	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return 0, 0, ErrInvalidToken
	}

	_, _ = q.Exec(ctx, `UPDATE personal_access_tokens SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), tokenID)
	return userID, tokenID, nil
}

func DeleteToken(ctx context.Context, q db.Querier, tokenID int64) error {
	_, err := q.Exec(ctx, `DELETE FROM personal_access_tokens WHERE id = $1`, tokenID)
	return err
}
