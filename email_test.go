package dim

import (
	"bytes"
	"testing"
)

func TestNewBaseEmailData(t *testing.T) {
	cfg := &EmailConfig{
		AppName:      "TestApp",
		PrimaryColor: "#ff0000",
		LogoURL:      "http://logo.url",
		SocialLinks:  `[{"name":"Twitter","url":"http://twitter.com"}]`,
	}

	data := NewBaseEmailData(cfg)

	if data.AppName != "TestApp" {
		t.Errorf("expected AppName TestApp, got %s", data.AppName)
	}
	if data.PrimaryColor != "#ff0000" {
		t.Errorf("expected PrimaryColor #ff0000, got %s", data.PrimaryColor)
	}
	if !data.LogoURL.Valid || data.LogoURL.Value != "http://logo.url" {
		t.Errorf("expected LogoURL http://logo.url, got %v", data.LogoURL)
	}
	if len(data.SocialLinks) != 1 {
		t.Errorf("expected 1 social link, got %d", len(data.SocialLinks))
	}
	if data.SocialLinks[0].Name != "Twitter" {
		t.Errorf("expected social link name Twitter, got %s", data.SocialLinks[0].Name)
	}
}

func TestNewMailMessage(t *testing.T) {
	to := []string{"test@example.com"}
	subject := "Test Subject"
	msg := NewMailMessage(to, subject)

	if len(msg.To) != 1 || msg.To[0] != to[0] {
		t.Errorf("expected To %v, got %v", to, msg.To)
	}
	if msg.Subject != subject {
		t.Errorf("expected Subject %s, got %s", subject, msg.Subject)
	}
	if msg.Headers == nil {
		t.Error("expected initialized Headers map")
	}
}

func TestNewMailerFromConfig_Null(t *testing.T) {
	cfg := &EmailConfig{
		Transport: "null",
		From:      "noreply@example.com",
	}
	var buf bytes.Buffer

	mailer, err := NewMailerFromConfig(cfg, &buf)
	if err != nil {
		t.Fatalf("failed to create null mailer: %v", err)
	}
	if mailer == nil {
		t.Fatal("expected mailer instance")
	}
}

func TestNewMailerFromConfig_SMTP(t *testing.T) {
	// Note: We're not testing actual connection here, just config mapping
	cfg := &EmailConfig{
		Transport:    "smtp",
		From:         "noreply@example.com",
		SMTPHost:     "localhost",
		SMTPPort:     1025,
		SMTPUsername: "user",
		SMTPPassword: "password",
	}
	var buf bytes.Buffer

	mailer, err := NewMailerFromConfig(cfg, &buf)
	if err != nil {
		t.Fatalf("failed to create smtp mailer: %v", err)
	}
	if mailer == nil {
		t.Fatal("expected mailer instance")
	}
}
