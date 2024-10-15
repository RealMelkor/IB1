package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"io"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"

	"IB1/db"
)

type user struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *user) GetEmail() string {
	return u.Email
}

func (u user) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *user) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func copyFile(src string, dst string) error {
	from, err := os.Open(src)
	if err != nil { return err }
	defer from.Close()
	to, err := os.Create(dst)
	if err != nil { return err }
	defer to.Close()
	_, err = io.Copy(to, from)
	return err
}

func Generate(domain string, email string, port string, www bool) (
						[]byte, []byte, error) {

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { return nil, nil, err }

	user := user{
		Email: email,
		key:   privateKey,
	}

	conf := lego.NewConfig(&user)

	conf.CADirURL = "https://acme-v02.api.letsencrypt.org/directory"
	conf.Certificate.KeyType = certcrypto.RSA4096

	client, err := lego.NewClient(conf)
	if err != nil { return nil, nil, err }

	err = client.Challenge.SetHTTP01Provider(
		http01.NewProviderServer("", port))
	if err != nil { return nil, nil, err }

	reg, err := client.Registration.Register(
		registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil { return nil, nil, err }
	user.Registration = reg

	domains := []string{domain}
	if www { domains = append(domains, "www." + domain) }

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil { return nil, nil, err }

	return certificates.Certificate, certificates.PrivateKey,
		db.UpdateConfig()
}
