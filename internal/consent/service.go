package consent

import (
	"context"
)

type Service interface {
	RecordConsent(ctx context.Context, userId *string, ipAddress, userAgent string, consentGiven bool, consentType string) (*ConsentRecord, error)
	GetLatestConsent(ctx context.Context, userId string) (*ConsentRecord, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) RecordConsent(ctx context.Context, userId *string, ipAddress, userAgent string, consentGiven bool, consentType string) (*ConsentRecord, error) {
	if consentType == "" {
		consentType = "analytics"
	}

	record := &ConsentRecord{
		UserId:       userId,
		IpAddress:    ipAddress,
		UserAgent:    userAgent,
		ConsentGiven: consentGiven,
		ConsentType:  consentType,
	}

	err := s.repo.CreateConsentRecord(ctx, record)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *service) GetLatestConsent(ctx context.Context, userId string) (*ConsentRecord, error) {
	return s.repo.GetLatestConsentByUserId(ctx, userId)
}
