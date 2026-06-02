package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveUser       = errors.New("user is inactive")
	ErrRefreshExpired     = errors.New("refresh token expired or revoked")
)

type AuditWriter interface {
	Login(ctx context.Context, userID uuid.UUID, ip, userAgent string)
	LoginFailed(ctx context.Context, username, ip, userAgent string)
}

type Service struct {
	pool   *pgxpool.Pool
	repo   *Repository
	jwt    *JWTIssuer
	audit  AuditWriter
	refTTL time.Duration
}

func NewService(pool *pgxpool.Pool, repo *Repository, jwt *JWTIssuer, audit AuditWriter, refTTL time.Duration) *Service {
	return &Service{pool: pool, repo: repo, jwt: jwt, audit: audit, refTTL: refTTL}
}

func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (*LoginResponse, string, error) {
	row, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.audit.LoginFailed(ctx, username, ip, userAgent)
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}
	if !row.IsActive {
		s.audit.LoginFailed(ctx, username, ip, userAgent)
		return nil, "", ErrInactiveUser
	}
	ok, err := VerifyPassword(password, row.PasswordHash)
	if err != nil || !ok {
		s.audit.LoginFailed(ctx, username, ip, userAgent)
		return nil, "", ErrInvalidCredentials
	}
	roles, deptIDs, secIDs, err := s.repo.GetUserRoles(ctx, row.ID)
	if err != nil {
		return nil, "", err
	}

	access, exp, err := s.jwt.Issue(row.ID, roles)
	if err != nil {
		return nil, "", err
	}
	refresh, refHash, err := generateRefreshToken()
	if err != nil {
		return nil, "", err
	}
	if err := s.repo.InsertRefreshToken(ctx, row.ID, refHash,
		time.Now().UTC().Add(s.refTTL), userAgent, ip); err != nil {
		return nil, "", err
	}

	s.audit.Login(ctx, row.ID, ip, userAgent)

	return &LoginResponse{
		AccessToken: access,
		ExpiresAt:   exp,
		User:        rowToUser(row, roles, deptIDs, secIDs),
	}, refresh, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, ip, userAgent string) (*LoginResponse, string, error) {
	hash := hashRefreshToken(refreshToken)
	rt, err := s.repo.GetActiveRefreshToken(ctx, hash)
	if err != nil {
		return nil, "", ErrRefreshExpired
	}
	if rt.RevokedAt != nil || rt.ExpiresAt.Before(time.Now().UTC()) {
		return nil, "", ErrRefreshExpired
	}
	row, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil || !row.IsActive {
		return nil, "", ErrInvalidCredentials
	}
	roles, deptIDs, secIDs, err := s.repo.GetUserRoles(ctx, row.ID)
	if err != nil {
		return nil, "", err
	}

	// Rotate: revoke old, issue new.
	if err := s.repo.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, "", err
	}
	access, exp, err := s.jwt.Issue(row.ID, roles)
	if err != nil {
		return nil, "", err
	}
	newRefresh, newHash, err := generateRefreshToken()
	if err != nil {
		return nil, "", err
	}
	if err := s.repo.InsertRefreshToken(ctx, row.ID, newHash,
		time.Now().UTC().Add(s.refTTL), userAgent, ip); err != nil {
		return nil, "", err
	}

	return &LoginResponse{
		AccessToken: access,
		ExpiresAt:   exp,
		User:        rowToUser(row, roles, deptIDs, secIDs),
	}, newRefresh, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	rt, err := s.repo.GetActiveRefreshToken(ctx, hashRefreshToken(refreshToken))
	if err != nil {
		return nil
	}
	return s.repo.RevokeRefreshToken(ctx, rt.ID)
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*User, error) {
	row, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles, deptIDs, secIDs, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	u := rowToUser(row, roles, deptIDs, secIDs)
	return &u, nil
}

func rowToUser(row *userRow, roles []string, deptIDs []uuid.UUID, secIDs []uuid.UUID) User {
	return User{
		ID:                 row.ID,
		PersonnelNumber:    row.PersonnelNumber,
		Username:           row.Username,
		Email:              row.Email,
		FullName:           row.FullName,
		BirthDate:          row.BirthDate,
		DepartmentID:       row.DepartmentID,
		SectionID:          row.SectionID,
		GradeID:            row.GradeID,
		PositionID:         row.PositionID,
		Specialization:     row.Specialization,
		TelegramID:         row.TelegramID,
		HiredAt:            row.HiredAt,
		Hobbies:            row.Hobbies,
		MustChangePassword: row.MustChangePassword,
		IsActive:           row.IsActive,
		Roles:              roles,
		ScopeDepartmentIDs: deptIDs,
		ScopeSectionIDs:    secIDs,
	}
}

func generateRefreshToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("read rand: %w", err)
	}
	tok := hex.EncodeToString(b)
	return tok, hashRefreshToken(tok), nil
}

func hashRefreshToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}
