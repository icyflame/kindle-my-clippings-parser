package notifier

import (
	"fmt"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"github.com/icyflame/kindle-my-clippings-parser/internal/utils"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"go.uber.org/zap"
)

type EmailSendgridNotifier struct {
	Logger    *zap.Logger
	APIKey    string
	Sender    Source
	Receivers []Source
}

func (e *EmailSendgridNotifier) Notify(clipping parser.Clipping) error {
	subject := "Today's Excerpt"
	from := mail.NewEmail(e.Sender.Name, e.Sender.Email)
	to := mail.NewEmail(e.Receivers[0].Name, e.Receivers[0].Email)

	plainTextContent, err := utils.MakePlaintextEmailFromClipping(clipping)
	if err != nil {
		return fmt.Errorf("could not construct plain text email from clipping %v > %w", clipping, err)
	}
	message := mail.NewSingleEmailPlainText(from, subject, to, plainTextContent)

	client := sendgrid.NewSendClient(e.APIKey)
	response, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("could not send email using sendgrid (%v) > %w", response, err)
	}

	e.Logger.Debug("sent email via sendgrid", zap.Any("response", response))

	return nil
}
