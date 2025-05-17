package mailer

import (
	"bytes"
	"embed"
	"time"

	"github.com/wneessen/go-mail"

	// Import the html/template and text template packages
	// these share the same package name ("template") we need disambiguate them
	// and alias them to ht and tt respectively
	ht "html/template"
	tt "text/template"
)

// hole our email templates
// a comment directive in the format `//go:embed <path>`
// IMMEDIATELY ABOVE it, which indicateds to go that we want to store
// the contents of the ./templates directory in the templateFS embedded filesystem variable

//go:embed "templates"
var templateFS embed.FS

// Mailer struct contains a mail.Client instance (used to connect to a SMTP server)
// and the sender information for your emails (the name and address you
// want the email to be from)
type Mailer struct {
	client *mail.Client
	sender string
}

// the Initialize function
// parameters: given SMTP server settings
//
//	with 5 seconds timeout
func New(host string, port int, username, password, sender string) (*Mailer, error) {
	client, err := mail.NewClient(
		host,
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithPort(port),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	mailer := &Mailer{
		client: client,
		sender: sender,
	}

	return mailer, nil
}

// the function to send a message to the SMTP server
// parameters: the recipient email address
//
//	the name of the file containing the templates
//	dynamic data for the templates
func (m *Mailer) Send(recipient string, templateFile string, data any) error {
	// parse the required template file from the embedded file system
	textTmpl, err := tt.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Execute the named template "subject", passing in the dynamic data and storing
	// the result in a bytes.Buffer variable.
	subject := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	// Execute the "plainBody" template and store the result
	// in the plainBody variable
	plainBody := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	// parse the required template file from the embedded file system
	htmlTmpl, err := ht.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Execute the "htmlBody" template and store the result
	// in the htmlBody variable
	htmlBody := new(bytes.Buffer)
	err = htmlTmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Initialize a new mail.Msg instance
	msg := mail.NewMsg()

	// set recipient
	err = msg.To(recipient)
	if err != nil {
		return err
	}

	// set sender
	err = msg.From(m.sender)
	if err != nil {
		return err
	}

	msg.Subject(subject.String())                                  // set subject
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())      // set plain-text body
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String()) // set html body with the AddAlternativeString method

	// passing in the message to send
	// open a connection to the SMTP server, sends the message the
	// closes the connection
	return m.client.DialAndSend(msg)
}
