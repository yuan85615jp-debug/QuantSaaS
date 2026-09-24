package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yuan85615jp-debug/QuantSaaS/internal/saas/store"
	"gorm.io/gorm"
)

var (
	ErrUserExists     = errors.New("user already exists")
	ErrInvalidCreds   = errors.New("invalid credentials")
	ErrUserNotFound   = errors.New("user not found")
	ErrRoleNotAllowed = errors.New("role not allowed for this endpoint")
)

// UserService handles registration and credential checks against the store.
type UserService struct {
	db   *store.DB
	auth *Service
}

func NewUserService(db *store.DB, authSvc *Service) *UserService {
	return &UserService{db: db, auth: authSvc}
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"` // optional; only "user" accepted publicly (admin/agent seeded)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token string      `json:"token"`
	User  *store.User `json:"user,omitempty"`
}

// Register creates a new user with role "user" (or explicit agent when seeded in tests/dev).
func (s *UserService) Register(req RegisterRequest) (*store.User, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%w: invalid email", ErrInvalidCreds)
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" {
		role = "user"
	}
	// Public register only allows user; agent/admin must be created via seed or admin path.
	if role != "user" && role != "agent" {
		role = "user"
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	u := store.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  name,
		Role:         role,
		Plan:         "free",
	}
	if err := s.db.Create(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return &u, nil
}

// Login validates credentials and returns a JWT + user (role from DB).
func (s *UserService) Login(req LoginRequest) (*TokenResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	var u store.User
	if err := s.db.Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCreds
		}
		return nil, err
	}
	if !CheckPassword(u.PasswordHash, req.Password) {
		return nil, ErrInvalidCreds
	}
	tok, err := s.auth.SignToken(u.ID, u.Role)
	if err != nil {
		return nil, err
	}
	return &TokenResponse{Token: tok, User: &u}, nil
}

// AgentLogin same as Login but requires role agent or admin (for LocalAgent).
func (s *UserService) AgentLogin(req LoginRequest) (*TokenResponse, error) {
	resp, err := s.Login(req)
	if err != nil {
		return nil, err
	}
	if resp.User.Role != "agent" && resp.User.Role != "admin" {
		return nil, ErrRoleNotAllowed
	}
	// Re-sign to ensure role is agent/admin in claims
	tok, err := s.auth.SignToken(resp.User.ID, resp.User.Role)
	if err != nil {
		return nil, err
	}
	resp.Token = tok
	return resp, nil
}
