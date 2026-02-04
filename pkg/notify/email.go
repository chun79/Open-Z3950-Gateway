package notify

import (
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
)

type EmailService struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewEmailService() *EmailService {
	return &EmailService{
		host:     os.Getenv("SMTP_HOST"),
		port:     os.Getenv("SMTP_PORT"),
		username: os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASS"),
		from:     os.Getenv("SMTP_FROM"),
	}
}

func (s *EmailService) SendOverdueNotice(to, patronName, bookTitle, dueDate string) error {
	if s.host == "" {
		slog.Warn("SMTP not configured, skipping email notice", "to", to)
		return nil
	}

	subject := "Subject: 📚 图书逾期提醒: " + bookTitle + "
"
	mime := "MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";

"
	body := fmt.Sprintf(`
		<h3>尊敬的 %s：</h3>
		<p>您借阅的图书 <strong>《%s》</strong> 已于 <strong>%s</strong> 到期。</p>
		<p>请尽快归还至图书馆，以免产生更多滞纳金。</p>
		<hr>
		<p>Open Z39.50 LSP 自动发送</p>
	`, patronName, bookTitle, dueDate)

	msg := []byte(subject + mime + body)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	err := smtp.SendMail(s.host+":"+s.port, auth, s.from, []string{to}, msg)
	if err != nil {
		slog.Error("failed to send email", "error", err)
		return err
	}

	slog.Info("overdue notice sent", "to", to, "book", bookTitle)
	return nil
}
