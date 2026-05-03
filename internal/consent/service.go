package consent

import (
	"context"
	"strings"
)

type Service interface {
	RecordConsent(ctx context.Context, userId *string, ipAddress, userAgent string, consentGiven bool, consentType string, documentVersion string, locale string, source string) (*ConsentRecord, error)
	GetLatestConsent(ctx context.Context, userId string) (*ConsentRecord, error)
	GetConsentStatus(ctx context.Context, userId string) (map[string]ConsentRecord, error)
	HasGrantedConsent(ctx context.Context, userId string, consentType string) (bool, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) RecordConsent(ctx context.Context, userId *string, ipAddress, userAgent string, consentGiven bool, consentType string, documentVersion string, locale string, source string) (*ConsentRecord, error) {
	consentType = normalizeConsentType(consentType)
	documentVersion = strings.TrimSpace(documentVersion)
	if documentVersion == "" {
		documentVersion = DefaultDocumentVersion
	}
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale != "ru" && locale != "en" {
		locale = "ru"
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "web"
	}

	record := &ConsentRecord{
		UserId:          userId,
		IpAddress:       ipAddress,
		UserAgent:       userAgent,
		ConsentGiven:    consentGiven,
		ConsentType:     consentType,
		DocumentVersion: documentVersion,
		Locale:          locale,
		Source:          source,
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

func (s *service) GetConsentStatus(ctx context.Context, userId string) (map[string]ConsentRecord, error) {
	return s.repo.GetLatestConsentStatusByUserId(ctx, userId)
}

func (s *service) HasGrantedConsent(ctx context.Context, userId string, consentType string) (bool, error) {
	record, err := s.repo.GetLatestConsentByUserIdAndType(ctx, userId, normalizeConsentType(consentType))
	if err != nil {
		return false, err
	}
	return record != nil && record.ConsentGiven, nil
}

func normalizeConsentType(consentType string) string {
	switch strings.TrimSpace(consentType) {
	case TypePersonalDataProcessing, TypeAnalyticsCookies, TypeMarketingAds, TypeSpecialCategoryHealth, TypePublicDistribution:
		return strings.TrimSpace(consentType)
	case "analytics":
		return TypeAnalyticsCookies
	default:
		return TypeAnalyticsCookies
	}
}
