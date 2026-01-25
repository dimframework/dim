package dim

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/atfromhome/goreus/pkg/mail"
)

// Export goreus types for convenience and compatibility
type Mailer = mail.Mailer
type Attachment = mail.Attachment

// MailMessage aliases goreus Message for backward compatibility
type MailMessage = mail.Message

// SocialLink represents a social media link for email footer
type SocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// BaseEmailData holds common data for all email templates
type BaseEmailData struct {
	AppName      string
	LogoURL      JsonNull[string]
	PrimaryColor string
	SupportEmail JsonNull[string]
	SupportURL   JsonNull[string]
	CompanyName  JsonNull[string]
	SocialLinks  []SocialLink
	Year         int
}

// NewBaseEmailData membuat BaseEmailData dari EmailConfig.
// Mengonversi string konfigurasi ke dalam tipe data yang siap digunakan oleh template.
// Secara otomatis mengisi tahun saat ini dan mem-parse JSON social links jika ada.
func NewBaseEmailData(cfg *EmailConfig) BaseEmailData {
	data := BaseEmailData{
		AppName:      cfg.AppName,
		PrimaryColor: cfg.PrimaryColor,
		Year:         time.Now().Year(),
		SocialLinks:  []SocialLink{},
	}

	if cfg.LogoURL != "" {
		data.LogoURL = NewJsonNull(cfg.LogoURL)
	}

	if cfg.SupportEmail != "" {
		data.SupportEmail = NewJsonNull(cfg.SupportEmail)
	}

	if cfg.SupportURL != "" {
		data.SupportURL = NewJsonNull(cfg.SupportURL)
	}

	if cfg.CompanyName != "" {
		data.CompanyName = NewJsonNull(cfg.CompanyName)
	}

	if cfg.SocialLinks != "" {
		var links []SocialLink
		if err := json.Unmarshal([]byte(cfg.SocialLinks), &links); err == nil {
			data.SocialLinks = links
		}
	}

	return data
}

// NewMailMessage membuat instance MailMessage baru dengan penerima dan subjek yang ditentukan.
// Wrapper untuk goreus Message struct.
func NewMailMessage(to []string, subject string) *MailMessage {
	return &MailMessage{
		To:      to,
		Subject: subject,
		Headers: make(map[string]string),
	}
}

// NewMailerFromConfig membuat instance Mailer berdasarkan EmailConfig yang diberikan.
// Menggunakan library goreus/pkg/mail di belakang layar.
//
// Note: Parameter 'output' saat ini diabaikan karena goreus NullMailer selalu menulis ke stdout.
func NewMailerFromConfig(cfg *EmailConfig, output io.Writer) (Mailer, error) {
	goreusCfg := &mail.Config{
		Transport:   cfg.Transport,
		FromAddress: cfg.From,
	}

	// Map SMTP Config
	goreusCfg.SMTP.Host = cfg.SMTPHost
	goreusCfg.SMTP.Port = cfg.SMTPPort
	goreusCfg.SMTP.Username = cfg.SMTPUsername
	goreusCfg.SMTP.Password = cfg.SMTPPassword

	// Map SES Config
	goreusCfg.SES.Region = cfg.SESRegion
	goreusCfg.SES.AccessKeyID = cfg.SESAccessKeyID
	goreusCfg.SES.SecretAccessKey = cfg.SESSecretAccessKey
	goreusCfg.SES.ConfigurationSet = cfg.SESConfigurationSet

	return mail.NewTransport(context.Background(), goreusCfg)
}
