package notify

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailConfig SMTP 邮件发送配置。
type EmailConfig struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	FromAddr string
	FromName string
}

// EmailSender 邮件发送器，支持 STARTTLS（端口 587）和 SSL（端口 465）。
type EmailSender struct {
	cfg EmailConfig
}

// NewEmailSender 创建邮件发送器实例。
func NewEmailSender(cfg EmailConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

// Send 发送纯文本邮件。
func (s *EmailSender) Send(to, subject, body string) error {
	if err := s.validate(); err != nil {
		return err
	}
	return s.send(to, subject, body, "text/plain")
}

// SendHTML 发送 HTML 格式邮件。
func (s *EmailSender) SendHTML(to, subject, html string) error {
	if err := s.validate(); err != nil {
		return err
	}
	return s.send(to, subject, html, "text/html")
}

// IsConfigured 检查邮件配置是否完整（SMTP 地址、发件人地址已设置且密码不是占位符）。
func (s *EmailSender) IsConfigured() bool {
	return s.cfg.SMTPHost != "" && s.cfg.FromAddr != "" &&
		!strings.Contains(s.cfg.Password, "your-")
}

// validate 检查 SMTP 配置是否有效。
func (s *EmailSender) validate() error {
	if s.cfg.SMTPHost == "" || s.cfg.FromAddr == "" {
		return fmt.Errorf("email not configured: missing SMTP host or from address")
	}
	return nil
}

// send 构建邮件头并选择 STARTTLS 或 SSL 方式发送。
func (s *EmailSender) send(to, subject, body, contentType string) error {
	header := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\n\r\n%s",
		s.cfg.FromName, s.cfg.FromAddr, to, subject, contentType, body)

	if s.cfg.SMTPPort == 465 {
		return s.sendSSL(to, []byte(header))
	}
	return s.sendSTARTTLS(to, []byte(header))
}

// sendSTARTTLS 使用 STARTTLS 方式发送（端口 587）。
func (s *EmailSender) sendSTARTTLS(to string, msg []byte) error {
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	return smtp.SendMail(addr, auth, s.cfg.FromAddr, []string{to}, msg)
}

// sendSSL 使用 SSL/TLS 直连方式发送（端口 465）。
func (s *EmailSender) sendSSL(to string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	tlsCfg := &tls.Config{ServerName: s.cfg.SMTPHost}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls connect: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(s.cfg.FromAddr); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return nil
}

