// gplus2others - Send Google+ activities to other networks
//
// Copyright 2011 The gplus2others Authors.  All rights reserved.
// Use of this source code is governed by the Simplified BSD
// license that can be found in the LICENSE file.

package gplus2others

import (
	"net/http"
	"time"

	"appengine"
	"appengine/urlfetch"
	"code.google.com/p/goauth2/oauth"
	plus "google.golang.org/api/plus/v1"
)

func config(host string) *oauth.Config {
	return &oauth.Config{
		ClientId:       appConfig.GoogleClientId,
		ClientSecret:   appConfig.GoogleClientSecret,
		Scope:          plus.PlusMeScope,
		AuthURL:        "https://accounts.google.com/o/oauth2/auth",
		TokenURL:       "https://accounts.google.com/o/oauth2/token",
		RedirectURL:    "http://" + host + "/oauth2callback",
		AccessType:     "offline",
		ApprovalPrompt: "force",
	}
}

func emptyTransport() *oauth.Transport {
	return transport(User{})
}

func loginGoogle(w http.ResponseWriter, r *http.Request) {
	tr := emptyTransport()
	c := appengine.NewContext(r)
	tr.Transport = &urlfetch.Transport{Context: c}
	urls := tr.AuthCodeURL("login")
	c.Debugf("Google AuthCodeURL: %s\n", urls)
	http.Redirect(w, r, urls, http.StatusFound)
}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	code := r.FormValue("code")
	tr := emptyTransport()
	tr.Transport = &urlfetch.Transport{Context: c}
	if _, err := tr.Exchange(code); err != nil {
		c.Debugf("tr: %v\n", tr.Token)
		serveError(c, w, err)
		return
	}
	// get info on the user
	httpClient := tr.Client()
	p, err := plus.New(httpClient)
	if err != nil {
		serveError(c, w, err)
		return
	}
	c.Debugf("get person")
	person, err := p.People.Get("me").Do()
	c.Debugf("get person:::::::: %v\n", err)
	if err != nil {
		serveError(c, w, err)
		return
	}
	cookie := &http.Cookie{Name: "userId", Value: person.Id, Domain: appConfig.AppDomain, Path: "/", MaxAge: 30000000 /* about a year */}
	http.SetCookie(w, cookie)

	user := loadUser(r, person.Id)

	user.GoogleAccessToken = tr.Token.AccessToken
	user.GoogleTokenExpiry = tr.Token.Expiry.UnixNano()
	user.GoogleRefreshToken = tr.Token.RefreshToken
	if user.Id == "" {
		user.Id = person.Id
		user.GoogleLatest = time.Now().UnixNano()

	}
	saveUser(r, &user)

	http.Redirect(w, r, "/", http.StatusFound)
}

func transport(user User) *oauth.Transport {
	return &oauth.Transport{
		Token:     &oauth.Token{AccessToken: user.GoogleAccessToken, RefreshToken: user.GoogleRefreshToken, Expiry: time.Unix(0, user.GoogleTokenExpiry)},
		Config:    config(appConfig.AppHost),
		Transport: &urlfetch.Transport{},
	}
}
