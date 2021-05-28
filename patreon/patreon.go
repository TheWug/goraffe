package patreon

import (
	"time"
	"errors"
	"encoding/json"
	"net/http"
	"fmt"
)

var BadLogin error = errors.New("Invalid or expired patreon auth token")

type Tier struct {
	Name              string
	ContributionCents int
}

type TierArray []Tier

type basicPatreonSession struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	Scope        string `json:"scope,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

type PatreonSession struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scope        string
	TokenType    string
}

func (p PatreonSession) MarshalJSON() ([]byte, error) {
	bp := basicPatreonSession{
		AccessToken: p.AccessToken,
		RefreshToken: p.RefreshToken,
		ExpiresAt: p.ExpiresAt.Format(time.RFC3339),
		Scope: p.Scope,
		TokenType: p.TokenType,
	}

	return json.Marshal(bp)
}

func (p *PatreonSession) UnmarshalJSON(j []byte) error {
	var bp basicPatreonSession
	err := json.Unmarshal(j, &bp)
	if err != nil {
		return err
	}

	var expires_at time.Time
	if bp.ExpiresAt == "" {
		expires_at = time.Now().Add(time.Duration(bp.ExpiresIn) * time.Second)
	} else {
		expires_at, err = time.Parse(time.RFC3339, bp.ExpiresAt)
	}

	if err != nil {
		return err
	}

	*p = PatreonSession{
		AccessToken: bp.AccessToken,
		RefreshToken: bp.RefreshToken,
		ExpiresAt: expires_at,
		Scope: bp.Scope,
		TokenType: bp.TokenType,
	}

	return nil
}

func GetCampaignTiers(p *PatreonSession) (TierArray, error) {
	return TierArray{}, nil
}
