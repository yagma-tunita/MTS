package notify

type Provider struct {
	Email *EmailSender
	SMS   SMSProvider
}

func NewProvider(emailCfg EmailConfig, smsCfg SMSConfig) *Provider {
	return &Provider{
		Email: NewEmailSender(emailCfg),
		SMS:   NewSMSProvider(smsCfg),
	}
}
