package users

import (
	"bytes"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
)

// SendMail sends an email based on the specified mail template (located in
// Config.MailTemplateDir or a subdirectory of it, depending on the value of
// Config.Internationalization) executed on the given data. The first line of
// the mail template will be used the email's subject. It must be followed by an
// empty line before the mail body starts.
//
// If this call fails with "x509: certificate signed by unknown authority" and
// the email server (operated by you) is using a self-signed certificate, you
// will need to add it to your server's list of trusted certificates.
func SendMail(request *http.Request, email, mailTemplate string, data interface{}) error {
	if !Config.SendEmails {
		Config.Log.Printf(`Requested email with template "%s" but sending is turned off`, mailTemplate)
		return nil // It's turned off.
	}

	// Determine template subdirectory.
	var subdirectory string
	if Config.Internationalization {
		if cookie, err := request.Cookie("lang"); err == nil {
			languageFormat := regexp.MustCompile("^[a-zA-Z]{2}(-[a-zA-Z]{2})?$")
			if languageFormat.MatchString(cookie.Value) {
				if dir, err := os.Stat(filepath.Join(Config.MailTemplateDir, cookie.Value)); err == nil {
					if dir.IsDir() {
						subdirectory = cookie.Value
					}
				}
			}
		}
	}

	// Render template.
	fileList := append([]string{mailTemplate}, Config.MailTemplateIncludes...)
	for index, file := range fileList {
		fileList[index] = filepath.Join(Config.MailTemplateDir, subdirectory, file)
	}
	tmpl, err := template.ParseFiles(fileList...)
	if err != nil {
		return fmt.Errorf(`Template "%s" could not be parsed: %s`, mailTemplate, err)
	}
	var text bytes.Buffer
	if err = tmpl.Execute(&text, data); err != nil {
		return fmt.Errorf(`Could not execute template "%s": %s`, mailTemplate, err)
	}

	// Maybe we'll use an external email function?
	if Config.SendEmail != nil {
		// Extract subject and body.
		split := regexp.MustCompile("(?ms)^(.*?)[\n\r]+(.*)$")
		fields := split.FindSubmatch(text.Bytes())
		if err = Config.SendEmail(email, string(fields[1]), string(fields[2])); err != nil {
			return fmt.Errorf("Error sending email with external code: %s", err)
		}
		return nil
	}

	// Prepare email body.
	linebreakFix := regexp.MustCompile(`\r?\n`)
	body := append(
		[]byte("From: "+Config.SenderName+" <"+Config.SenderEmail+">\r\n"+
			"To: "+email+"\r\n"+
			"Subject: "),
		linebreakFix.ReplaceAll(text.Bytes(), []byte("\r\n"))...,
	)

	// Send email.
	auth := smtp.PlainAuth("", Config.SMTPUsername, Config.SMTPPassword, Config.SMTPHostname)
	err = smtp.SendMail(
		fmt.Sprintf("%s:%d", Config.SMTPHostname, Config.SMTPPort),
		auth,
		Config.SenderEmail,
		[]string{email},
		body,
	)
	if err != nil {
		return fmt.Errorf("Could not send email (%s): %s", mailTemplate, err)
	}
	return nil
}
