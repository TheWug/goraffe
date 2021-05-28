package store

import (
	"database/sql"
)

var connection *sql.DB

func Init(dburl string) error {
	c, err := sql.Open("postgres", dburl)
	if err != nil {
		return err
	}

	connection = c
	return nil
}

type Raffle struct {
}

type Entry struct {
}

type Score struct {
}

func (this *Raffle) Status(user_id int) (*Entry, error) {
	return nil, nil
}

func (this *Raffle) Enter(user_id int, display string) (bool, error) {
	return false, nil
}

func (this *Raffle) Withdraw(user_id int, display string) (bool, error) {
	return false, nil
}

func (this *Raffle) Disqualify(user_id int) (bool, error) {
	return false, nil
}

func (this *Raffle) Undisqualify(user_id int) (*Entry, error) {
	return nil, nil
}

func (this *Raffle) Open() (bool, error) {
	return false, nil
}

func (this *Raffle) Close() (bool, error) {
	return false, nil
}

func (this *Raffle) Cancel() (bool, error) {
	return false, nil
}

func (this *Raffle) Draw() (*Entry, error) {
	return nil, nil
}
