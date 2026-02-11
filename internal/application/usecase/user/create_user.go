package user

import (
	"context"
	"fmt"

	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserInput struct {
	Email    string
	Password string
	Name     string
}

type CreateUserOutput struct {
	User *entity.User `json:"user"`
}

type CreateUserUseCase struct {
	userRepo repository.UserRepository
}

func NewCreateUserUseCase(userRepo repository.UserRepository) *CreateUserUseCase {
	return &CreateUserUseCase{userRepo: userRepo}
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) (*CreateUserOutput, error) {
	// Check if email already exists
	existing, _ := uc.userRepo.FindByEmail(ctx, input.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := entity.NewUser(input.Email, string(hash), input.Name)

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &CreateUserOutput{User: user}, nil
}
