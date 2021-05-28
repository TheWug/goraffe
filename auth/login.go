package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/thewug/goraffe/patreon"
)

type basicSession struct {
	Patreon      patreon.PatreonSession `json:"patreon,omitempty"`
	SessionDate  string `json:"session_date,omitempty"`
}

type Session struct {
	Patreon      patreon.PatreonSession
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

var aes_key []byte = nil

func Init(key, coder string) {
	mac := hmac.New(sha256.New, []byte(coder))
	mac.Write([]byte(key))
	aes_key = mac.Sum(nil)[0:32]
}

func getEncryptionKey() []byte {
	if aes_key == nil {
		panic("call Init first!")
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

const sessionCookieName string = "session"

func Get(req *http.Request) *Session {
	// fetch the session cookie and bail if it's unset.
	session_blob, err := req.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}

	// decrypt the session cookie into a session and bail if there's an error of any kind.
	var session Session
	err = DecryptAndValidate(session_blob.Value, &session)
	if err != nil {
		// XXX log this error as it is likely either developer error or evidence of abuse
		return nil
	}

	// check if session has expired and bail if so
	if time.Now().Sub(session.SessionDate) > one_week {
		return nil
	}

	return &session
}

func Put(w http.ResponseWriter, s *Session) {
	s.Update()
	value, err := EncryptAndSign(*s)
	if err != nil {
		// XXX log this
		return
	}

	var c http.Cookie
	c.Name = sessionCookieName
	c.Value = value
	c.Expires = s.SessionDate.Add(one_week)
//	c.Secure = true
	c.HttpOnly = true
	c.SameSite = http.SameSiteStrictMode

	http.SetCookie(w, &c)
}

func Delete(w http.ResponseWriter) {
	var c http.Cookie
	c.Name = sessionCookieName
	c.Value = ""
	c.MaxAge = -1
	c.HttpOnly = true
	c.SameSite = http.SameSiteStrictMode

	http.SetCookie(w, &c)
}
