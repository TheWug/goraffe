package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
	"io"
	"fmt"
	"errors"
	"crypto/aes"
	"crypto/sha256"
	"crypto/hmac"
	"crypto/rand"
	"crypto/cipher"
)

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

type basicSession struct {
	Patreon      PatreonSession `json:"patreon,omitempty"`
	SessionDate  string `json:"session_date,omitempty"`
}

type Session struct {
	Patreon      PatreonSession
	SessionDate  time.Time
}

func (s *Session) Update() {
	s.SessionDate = time.Now()
}

func (s Session) MarshalJSON() ([]byte, error) {
	bs := basicSession{
		Patreon:      s.Patreon,
		SessionDate:  s.SessionDate.Format(time.RFC3339),
	}

	return json.Marshal(bs)
}

func (s *Session) UnmarshalJSON(j []byte) error {
	var bs basicSession
	err := json.Unmarshal(j, &bs)
	if err != nil {
		return err
	}

	session_date, err := time.Parse(time.RFC3339, bs.SessionDate)
	if err != nil {
		return err
	}

	*s = Session{
		Patreon:      bs.Patreon,
		SessionDate:  session_date,
	}

	return nil
}

type PatreonState struct {
	ReturnTo string `json:"return_to"`
	IV       string `json:"iv"`
}

const aes_keystring string = "One may opine for an eternity about what makes a password secure, but in reality, it's all about luck, and setting oneself up to be lucky. 69 420 funny numb3r haha!@$%^&"
const aes_keycoder  string = "xxxxxxxxxx69.420.000000000000000"
var aes_key []byte = nil

func getEncryptionKey() []byte {
	if aes_key == nil {
		mac := hmac.New(sha256.New, []byte(aes_keycoder))
		mac.Write([]byte(aes_keystring))
		aes_key = mac.Sum(nil)[0:32]
	}
	return aes_key
}

func getCipher() cipher.AEAD {
	block, err := aes.NewCipher(getEncryptionKey())
	if err != nil {
		panic(err.Error())
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	return gcm
}

var gcm_cipher cipher.AEAD = getCipher()

func EncryptAndSign(obj interface{}) (string, error) {
	var o string
	plaintext, err := json.Marshal(obj)
	if err != nil {
		return o, err
	}

	nonce := make([]byte, 12)
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return o, err
	}

	var ciphertext []byte
	ciphertext = append(ciphertext, nonce...)
	ciphertext = gcm_cipher.Seal(ciphertext, nonce, plaintext, nil)

	o = base64.RawURLEncoding.EncodeToString(ciphertext)
	return o, nil
}

func DecryptAndValidate(es string, obj interface{}) error {
	ciphertext, err := base64.RawURLEncoding.DecodeString(es)
	if err != nil {
		return err
	}

	if len(ciphertext) < gcm_cipher.NonceSize() {
		return errors.New("invalid ciphertext size")
	}

	nonce := ciphertext[:gcm_cipher.NonceSize()]
	ciphertext = ciphertext[gcm_cipher.NonceSize():]

	plaintext, err := gcm_cipher.Open(ciphertext[:0], nonce, ciphertext, nil)

	err = json.Unmarshal(plaintext, obj)
	if err != nil {
		return err
	}

	return nil
}

var one_week time.Duration = time.Hour * 24 * 7

func Get(req *http.Request) *Session {
	// fetch the session cookie and bail if it's unset.
	session_blob, err := req.Cookie("session")
	if err != nil {
		fmt.Println("no cookie at all")
		return nil
	}

	// decrypt the session cookie into a session and bail if there's an error of any kind.
	var session Session
	err = DecryptAndValidate(session_blob.Value, &session)
	if err != nil {
		fmt.Println("cookie decrypted badly: ", err.Error())
		// XXX log this error as it is likely either developer error or evidence of abuse
		return nil
	}

	// check if session has expired and bail if so
	if time.Now().Sub(session.SessionDate) > one_week {
		fmt.Println("cookie is expired")
		return nil
	}

	return &session
}

func Put(w http.ResponseWriter, s *Session) {
	s.Update()
	value, err := EncryptAndSign(*s)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		// XXX log this
		return
	}

	var c http.Cookie
	c.Name = "session"
	c.Value = value
	c.Expires = s.SessionDate.Add(one_week)
//	c.Secure = true
	c.HttpOnly = true
	c.SameSite = http.SameSiteStrictMode

	http.SetCookie(w, &c)
}
