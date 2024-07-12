package env

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

var Version string = "unset"

type Environment struct {
	SendgridAPIKey string `envconfig:"SENDGRID_API_KEY" required:"true"`
	SenderName     string `envconfig:"sender_name"`
	SenderEmail    string `envconfig:"sender_email"`
	ReceiverName   string `envconfig:"receiver_name"`
	ReceiverEmail  string `envconfig:"receiver_email"`
}

// Process ...
func Process() (Environment, error) {
	godotenv.Load()
	var environ Environment
	err := envconfig.Process("", &environ)
	if err != nil {
		return Environment{}, fmt.Errorf("could not process environment variables > %w", err)
	}

	return environ, nil
}
