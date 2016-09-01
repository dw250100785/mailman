package mail

import (
	"bytes"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/russross/blackfriday"
	"github.com/xuqingfeng/mailman/account"
	"github.com/xuqingfeng/mailman/smtp"
	"github.com/xuqingfeng/mailman/util"
	"gopkg.in/gomail.v2"
)

type Mail struct {
	Subject             string   `json:"subject"`
	To                  []string `json:"to"`
	Cc                  []string `json:"cc"`
	From                string   `json:"from"`
	Priority            bool     `json:"priority"`
	Body                string   `json:"body"`
	Token               string   `json:"token"`
	AttachmentFileNames []string `json:"attachmentFileNames"`
}

func SendMail(mail Mail) error {

	account, err := account.GetAccountInfo(mail.From)
	if err != nil {
		return err
	}
	SMTPServer, err := smtp.GetSMTPServer(mail.From)
	if err != nil {
		return err
	}
	m := gomail.NewMessage()
	m.SetHeader("Subject", mail.Subject)
	m.SetHeader("To", mail.To...)

	// Cc is empty
	if len(mail.Cc) > 0 {
		// multiple cc
		m.SetHeader("Cc", mail.Cc...)
	}
	// todo with name - SetAddressHeader
	//m.SetAddressHeader("Cc", mail.Cc[0].Email, mail.Cc[0].Name)
	m.SetHeader("From", account.Email)

	// priority
	if mail.Priority {
		m.SetHeader("X-Priority", "1")
	}

	// attachment
	tmpDir := util.GetTmpDir()
	attachmentDir := filepath.Join(tmpDir, mail.Token)
	if len(mail.AttachmentFileNames) > 0 {
		for _, v := range mail.AttachmentFileNames {
			attachmentPath := filepath.Join(attachmentDir, v)
			m.Attach(attachmentPath)
		}
	}

	content := ParseMailContent(mail.Body)

	m.SetBody("text/html", content)

	port, err := strconv.Atoi(SMTPServer.Port)
	if err != nil {
		return err
	}
	d := gomail.NewDialer(SMTPServer.Server, port, account.Email, account.Password)
	if err = d.DialAndSend(m); err != nil {
		util.FileLog.Error(err.Error())
		return err
	}
	return nil
}

func ParseMailContent(body string) (content string) {

	// markdown parse
	parsedContent := blackfriday.MarkdownCommon([]byte(body))
	content = string(parsedContent)

	// credit: https://github.com/leemunroe/responsive-html-email-template
	// if responsive.html changes, run `go-bindata .`
	mailTemplateContent, err := Asset(util.MailTemplateType + ".html")
	if err != nil {
		util.FileLog.Warn(err.Error())
		return
	} else {
		type mailContent struct {
			Content string
		}
		tpl, err := template.New("mail").Parse(string(mailTemplateContent))
		if err != nil {
			util.FileLog.Warn(err.Error())
			return
		} else {
			var buf bytes.Buffer
			mc := mailContent{
				content,
			}
			err = tpl.Execute(&buf, mc)
			if err != nil {
				util.FileLog.Warn(err.Error())
				return
			} else {
				content = buf.String()
			}
		}
	}

	return
}
