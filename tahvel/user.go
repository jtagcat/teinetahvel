package tahvel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jtagcat/util/std"
)

type (
	Tahvel struct {
		Session string
	}

	User struct {
		IDCode                  string `json:"name"`
		UserId                  int    `json:"user"`
		PersonId                int    `json:"person"`
		FullName                string
		Roles                   []UserRole `json:"users"`
		SessionTimeoutInSeconds int
	}
	UserRole struct {
		Id           int
		SchoolCode   string
		Role         string
		StudentGroup string
	}
)

// phone format: +37255555555
func AuthMid(ctx context.Context, idCode, phone string, authConfirmationCode chan<- string) (*Tahvel, error) {
	reqData := struct {
		IdCode string `json:"idcode"`
		Phone  string `json:"mobileNumber"`
	}{
		IdCode: idCode,
		Phone:  phone,
	}
	reqDataJ, err := json.Marshal(&reqData)
	if err != nil {
		return nil, fmt.Errorf("marshalling request json body: %w", err)
	}

	resp, err := std.PostWithContext(ctx, http.DefaultClient, baseURL+"/hois_back/mIdLogin", "application/json;charset=utf-8", bytes.NewReader(reqDataJ))
	if err != nil {
		return nil, fmt.Errorf("starting flow: %w", err)
	}

	var respCodeJ struct {
		ChallengeID string `json:"challengeID"`
	}

	respCodeB, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response code: %w", err)
	}
	if err := json.Unmarshal(respCodeB, &respCodeJ); err != nil {
		return nil, fmt.Errorf("decoding response code: %w", err)
	}

	authConfirmationCode <- respCodeJ.ChallengeID
	close(authConfirmationCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/hois_back/mIdAuthentication", nil)
	if err != nil {
		return nil, fmt.Errorf("crafting request: %w", err)
	}

	req.Header.Add("Authorization", resp.Header["Authorization"][0])

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("bad status")
	}

	tahvelSession := std.CookieByKey(resp.Cookies(), "SESSION")
	if tahvelSession == "" {
		return nil, fmt.Errorf("authentication response did not include token")
	}

	return &Tahvel{Session: tahvelSession}, nil
}

func (t *Tahvel) GetUser(ctx context.Context) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/hois_back/user", nil)
	if err != nil {
		return nil, fmt.Errorf("crafting request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "SESSION", Value: t.Session})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("bad status")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("decoding request: %w", err)
	}

	return &user, nil
}

func (t *Tahvel) Logout(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/hois_back/logout", nil)
	if err != nil {
		return fmt.Errorf("crafting request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "SESSION", Value: t.Session})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}

	if !(resp.StatusCode >= 400) {
		return fmt.Errorf("bad status")
	}

	return nil
}
