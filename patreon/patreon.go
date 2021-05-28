package patreon

import (
	"time"
	"errors"
	"encoding/json"
	"net/http"
	"fmt"
	"strings"

	"github.com/jmoiron/jsonq"
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

type PatreonUser struct {
	Id int
	FullName string
	CampaignId int
}

func GetTitleAndTiers(p *PatreonSession) (string, TierArray, error) {
	req, err := http.NewRequest("GET", "https://www.patreon.com/api/oauth2/v2/identity?include=campaign,campaign.tiers&fields%5Btier%5D=title,amount_cents&fields%5Buser%5D=full_name", nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", p.AccessToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}

	data := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", nil, err
	}
	jq := jsonq.NewQuery(data)
	v, err := jq.Interface("errors")
	if v != nil {
		return "", nil, errors.New("API call failed")
	}

	var vanity string
	var ta TierArray

	vanity, err = jq.String("data", "attributes", "full_name")
	if err != nil {
		return "", nil, err
	}

	included, err := jq.ArrayOfObjects("included")
	if err == nil {
		lc := func(s string, e error) string { return strings.ToLower(s) }
		for _, object := range included {
			jq2 := jsonq.NewQuery(object)
			if lc(jq2.String("type")) == "tier" {
				title, err := jq2.String("attributes", "title")
				if err != nil {
					return "", nil, err
				}
				amount, err := jq2.Int("attributes", "amount_cents")
				if err != nil {
					return "", nil, err
				}
				ta = append(ta, Tier{
					ContributionCents: amount,
					Name: title,
				})
			}
		}
	}
	return vanity, ta, nil
}
