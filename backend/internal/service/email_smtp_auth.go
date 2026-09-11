package service

import (
	"errors"
	"net"
	"net/smtp"
	"strings"
)

// smtpLoginAuth implements the AUTH LOGIN challenge/response flow used by
// Microsoft 365 and some SMTP relays that do not accept AUTH PLAIN.
type smtpLoginAuth struct {
	username string
	password string
	host     string
	step     int
}

func (a *smtpLoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if server.Name != a.host {
		return "", nil, errors.New("smtp server name mismatch")
	}
	if !server.TLS && !isLocalSMTPHost(server.Name) {
		return "", nil, errors.New("smtp authentication requires TLS")
	}
	return "LOGIN", nil, nil
}

func (a *smtpLoginAuth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch a.step {
	case 0:
		a.step++
		return []byte(a.username), nil
	case 1:
		a.step++
		return []byte(a.password), nil
	default:
		return nil, errors.New("smtp server requested too many AUTH LOGIN challenges")
	}
}

func isLocalSMTPHost(host string) bool {
	host = strings.TrimSpace(strings.TrimSuffix(host, "."))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// smtpAuthMechanism picks LOGIN when the relay advertises it. Microsoft 365
// commonly advertises LOGIN/XOAUTH2 and rejects the PLAIN mechanism.
func smtpAuthMechanism(extension string) string {
	hasPlain := false
	for _, mechanism := range strings.Fields(extension) {
		mechanism = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(mechanism)), "=")
		switch mechanism {
		case "LOGIN":
			return "LOGIN"
		case "PLAIN":
			hasPlain = true
		}
	}
	if hasPlain {
		return "PLAIN"
	}
	return "PLAIN"
}

func authenticateSMTP(client *smtp.Client, config *SMTPConfig) error {
	advertised, mechanisms := client.Extension("AUTH")
	if advertised && smtpAuthMechanism(mechanisms) == "LOGIN" {
		return client.Auth(&smtpLoginAuth{
			username: config.Username,
			password: config.Password,
			host:     config.Host,
		})
	}
	return client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host))
}
