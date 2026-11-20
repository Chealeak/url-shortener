package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"
)

const (
	aliasLength           = 6
	aliasGenerateAttempts = 5
)

var (
	ErrEmptyURL   = errors.New("empty url")
	ErrInvalidURL = errors.New("invalid url")
	ErrURLExists  = errors.New("url exists")
)

type Storage interface {
	SaveUrl(ctx context.Context, urlToSave string, alias string) (int64, error)
}

type Service struct {
	storage Storage
}

func New(storage Storage) *Service {
	return &Service{storage: storage}
}

func (s *Service) SaveUrl(ctx context.Context, rawUrl, alias string) (string, error) {
	if err := validateUrl(rawUrl); err != nil {
		return "", err
	}

	if alias != "" {
		return s.save(ctx, rawUrl, alias)
	}

	var err error
	for range aliasGenerateAttempts {
		alias = random.NewRandomString(aliasLength)
		alias, err = s.save(ctx, rawUrl, alias)
		if errors.Is(err, ErrURLExists) {
			continue
		}
		return alias, err
	}

	return "", ErrURLExists
}

func (s *Service) save(ctx context.Context, rawURL, alias string) (string, error) {
	if _, err := s.storage.SaveUrl(ctx, rawURL, alias); err != nil {
		if errors.Is(err, storage.ErrUrlExists) {
			return "", ErrURLExists
		}
		return "", fmt.Errorf("save url: %w", err)
	}

	return alias, nil
}

func validateUrl(rawURL string) error {
	if rawURL == "" {
		return ErrEmptyURL
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}

	return nil
}
