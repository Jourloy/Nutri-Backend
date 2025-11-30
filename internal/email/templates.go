package email

import "fmt"

// EmailTemplate представляет шаблон письма
type EmailTemplate struct {
	Subject string
	Body    string
}

// EmailTemplates содержит все шаблоны писем с локализацией
type EmailTemplates struct {
	VerificationCode map[string]EmailTemplate
}

// NewEmailTemplates создает экземпляр шаблонов писем
func NewEmailTemplates() *EmailTemplates {
	return &EmailTemplates{
		VerificationCode: map[string]EmailTemplate{
			"ru": {
				Subject: "Код подтверждения email - Nutri",
				Body: `Здравствуйте!

Ваш код подтверждения для привязки email к аккаунту Nutri:

%s

Код действителен 15 минут.

Если вы не запрашивали этот код, проигнорируйте это письмо.

С уважением,
Команда Nutri`,
			},
			"en": {
				Subject: "Email Verification Code - Nutri",
				Body: `Hello!

Your verification code to link your email to your Nutri account:

%s

The code is valid for 15 minutes.

If you did not request this code, please ignore this email.

Best regards,
Nutri Team`,
			},
		},
	}
}

// GetVerificationCodeTemplate возвращает шаблон письма с кодом подтверждения
func (t *EmailTemplates) GetVerificationCodeTemplate(locale string, code string) (subject string, body string) {
	template, ok := t.VerificationCode[locale]
	if !ok {
		// Fallback to Russian if locale not found
		template = t.VerificationCode["ru"]
	}

	return template.Subject, fmt.Sprintf(template.Body, code)
}
