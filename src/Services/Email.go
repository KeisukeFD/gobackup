package Services

import (
	"bytes"
	"errors"
	"fmt"
	"gobackup/src/Model"
	"gobackup/src/Utils"
	"math"
	"net/smtp"
	"time"
)

type EmailServer struct {
	Host     string
	Port     int
	MaxRetry int
	Password string
}

type Email struct {
	From    string
	To      string
	Subject string
	Body    string
}

func NewEmailServer(config *Model.Config) (*EmailServer, error) {
	server := &EmailServer{}
	if config.BackupConfig.Email.Host == "" {
		return nil, errors.New("email host must be valid")
	}
	server.Host = config.BackupConfig.Email.Host
	server.Port = config.BackupConfig.Email.Port
	server.MaxRetry = config.BackupConfig.Email.MaxTry
	server.Password = config.BackupConfig.Email.Password
	return server, nil
}

func (e *EmailServer) Send(email *Email) error {
	var err error
	for i := 1; i <= e.MaxRetry; i++ {
		err := sendEmail(e.Host, e.Port, email.From, e.Password, email.To, email.Body)
		if err == nil {
			break
		}
		Utils.GetLogger().Debug(fmt.Sprintf("Error sending email, try %d/%d (%s)", i, e.MaxRetry, err))
		time.Sleep(time.Duration(math.Pow(3, float64(i))) * time.Second)
	}
	return err
}

func sendEmail(host string, port int, from string, password string, to string, body string) error {
	auth := smtp.PlainAuth("", from, password, host)
	c, err := smtp.Dial(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Hello("localhost"); err != nil {
		return err
	}
	err = c.Mail(from)
	err = c.Rcpt(to)
	if password != "" {
		if err = c.Auth(auth); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(bytes.NewBufferString(body).Bytes())
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
