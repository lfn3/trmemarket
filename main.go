package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"io"
	"io/ioutil"

	"github.com/mrjones/oauth"

	"github.com/codegangsta/martini"
	"github.com/martini-contrib/sessions"
)

type Config struct {
	ApiBaseUrl string
	OAuthCfg OAuthCfg
}

type OAuthCfg struct {
	AdditionalRequestParams map[string]string
	ConsumerKey             string
	ConsumerSecret          string
	CallbackUrl             string
	ServiceProvider         oauth.ServiceProvider
}

type TradeMePagable struct {
	TotalCount int
	Page int
	PageSize int
	List []json.RawMessage
}

var config *Config
var consumer *oauth.Consumer
var outstandingTokens map[string]*oauth.RequestToken

func main() {
	file, err := os.Open("./config.json")

	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(file)
	config = &Config{}
	decoder.Decode(&config)

	file.Close()

	outstandingTokens = make(map[string]*oauth.RequestToken, 100)

	consumer = oauth.NewConsumer(config.OAuthCfg.ConsumerKey, config.OAuthCfg.ConsumerSecret, config.OAuthCfg.ServiceProvider)

	consumer.AdditionalParams = config.OAuthCfg.AdditionalRequestParams

	m := martini.Classic()

	store := sessions.NewCookieStore([]byte("secret123"))
	m.Use(sessions.Sessions("my_session", store))

	m.Get("/", func() string {
		return "Hello world!"
	})
	m.Get("/attach", LinkToTradeMe)
	m.Get("/callback", func(req *http.Request, session sessions.Session) string {
		urlParams := req.URL.Query()

		tokenString := strings.Join(urlParams["oauth_token"], "")
		verificationString := strings.Join(urlParams["oauth_verifier"], "")

		reqToken := outstandingTokens[tokenString]

		accessToken, err := consumer.AuthorizeToken(reqToken, verificationString)

		if err != nil {
			log.Println(err)
		}

		session.Set("accessToken", accessToken.Token)
		session.Set("accessTokenSecret", accessToken.Secret)

		return ("<p> Oauth token: " + accessToken.Token + "</p>")
	})
	m.Get("/fav/sellers", LinkToTradeMe, func(rw http.ResponseWriter, session sessions.Session) {
		res, err := consumer.Get(config.ApiBaseUrl + "Favourites/Sellers.json", nil, GetAccessToken(session))

		if (err != nil) {
			log.Println(err)
		}

		bytes, err := ioutil.ReadAll(res.Body)
		if (err != nil) {
			log.Println(err)
		}

		favList := TradeMePagable{}
		err = json.Unmarshal(bytes, &favList)
		if (err != nil) {
			log.Println(err)
		}

		_, err = io.Copy(rw, res.Body)

		if (err != nil) {
			log.Println(err)
		}
	})

	m.Run()
}

func GetAccessToken (session sessions.Session) *oauth.AccessToken {
	return &oauth.AccessToken {
		Token: session.Get("accessToken").(string),
		Secret: session.Get("accessTokenSecret").(string),
	}
}

func LinkToTradeMe(rw http.ResponseWriter, req *http.Request, session sessions.Session) {
	if session.Get("accessToken") != nil {
		return
	}

	requestToken, loginUrl, err := consumer.GetRequestTokenAndUrl(config.OAuthCfg.CallbackUrl)

	outstandingTokens[requestToken.Token] = requestToken

	if err != nil {
		log.Println(err)
	}

	http.Redirect(rw, req, loginUrl, 302)
}
