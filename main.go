package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"net/http"

	"github.com/mrjones/oauth"
	"github.com/codegangsta/martini"
)

type Config struct {
	OAuthCfg OAuthCfg
}

type OAuthCfg struct {
	AdditionalRequestParams map[string]string
	ConsumerKey             string
	ConsumerSecret          string
	CallbackUrl				string
	ServiceProvider 		oauth.ServiceProvider
}

func main() {
	file, err := os.Open("./config.json")

	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(file)
	config := &Config{}
	decoder.Decode(&config)

	file.Close()

	outstandingTokens := make(map[string]*oauth.RequestToken, 100)

	consumer := oauth.NewConsumer(config.OAuthCfg.ConsumerKey, config.OAuthCfg.ConsumerSecret, config.OAuthCfg.ServiceProvider)

	consumer.AdditionalParams = config.OAuthCfg.AdditionalRequestParams

  	m := martini.Classic()

	m.Get("/", func() string {
		return "Hello world!"
	})
	m.Get("/attach", func(res http.ResponseWriter, req *http.Request) {
		requestToken, loginUrl, err := consumer.GetRequestTokenAndUrl(config.OAuthCfg.CallbackUrl)

		outstandingTokens[requestToken.Token] = requestToken

		if err != nil {
			log.Fatal(err)
		}

		http.Redirect(res, req, loginUrl, 302)
	})
	m.Get("/callback", func(req *http.Request) string {
		urlParams := req.URL.Query()

		tokenString := strings.Join(urlParams["oauth_token"], "")
		verificationString := strings.Join(urlParams["oauth_verifier"], "")

		reqToken := outstandingTokens[tokenString]

		accessToken, err := consumer.AuthorizeToken(reqToken, verificationString)

		if err != nil {
			log.Fatal(err)
		}

		return ("<p> Oauth token: " + accessToken.Token + "</p>")
	})
	m.Run()
}
