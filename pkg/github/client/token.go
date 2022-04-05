/*
Copyright 2022 The Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

func newPersonalTokenSource(token string) oauth2.TokenSource {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return oauth2.ReuseTokenSource(nil, ts)
}

type installationTokenSource struct {
	appID          int64
	appPrivateKey  *rsa.PrivateKey
	installationID string

	BaseURL string
}

func newInstallationTokenSource(baseURL string, appID int64, appKey, installationID string) (oauth2.TokenSource, error) {
	f, err := os.ReadFile(filepath.Clean(appKey))
	if err != nil {
		return nil, fmt.Errorf("invalid app private key file: %v", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(f)
	if err != nil {
		return nil, fmt.Errorf("unable to parse app private key: %v", err)
	}

	ts := &installationTokenSource{
		BaseURL:        baseURL,
		appID:          appID,
		appPrivateKey:  privateKey,
		installationID: installationID,
	}

	tok, err := ts.Token()
	if err != nil {
		return nil, err
	}

	return oauth2.ReuseTokenSource(tok, ts), nil
}

type installationToken struct {
	Token        string                         `json:"token"`
	ExpiresAt    time.Time                      `json:"expires_at"`
	Permissions  github.InstallationPermissions `json:"permissions,omitempty"`
	Repositories []github.Repository            `json:"repositories,omitempty"`
}

func (ts *installationTokenSource) Token() (*oauth2.Token, error) {
	iss := time.Now().Add(-30 * time.Second).Truncate(time.Second)
	exp := iss.Add(2 * time.Minute)
	claims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(iss),
		ExpiresAt: jwt.NewNumericDate(exp),
		Issuer:    strconv.FormatInt(ts.appID, 10),
	}

	tokenJWT, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(ts.appPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("could not sign jwt: %s", err)
	}

	u := strings.TrimRight(ts.BaseURL, "/") + "/app/installations/" + ts.installationID + "/access_tokens"
	hc := oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tokenJWT}))
	res, err := hc.Post(u, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("unable create app installations access_tokens: %v", err)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 201 {
		return nil, errors.New("unable create app installations access_tokens. got non 2xx HTTP response code")
	}

	var it installationToken
	if err := json.NewDecoder(res.Body).Decode(&it); err != nil {
		return nil, fmt.Errorf("unable to decode app installations access_tokens response: %v", err)
	}

	return &oauth2.Token{
		AccessToken: it.Token,
		TokenType:   "bearer",
		Expiry:      it.ExpiresAt,
	}, nil
}
