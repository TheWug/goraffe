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
	Id        string
	Display   string
	Owner     int
	Tiers   []int32
	Timestamp int64
	IsOpen    bool
}

type Entry struct {
	RaffleId     string
	UserId       int
	Entered      bool
	Disqualified bool
	Name         string
}

type Score struct {
	RaffleId      string
	UserId        int
	Name          string
	Score         float64
	LifetimeScore float64
}

func (this *Raffle) Status(user_id int) (*Entry, error) {
	var e Entry
	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		row := tx.QueryRow("select id, user_id, display, entered, disqualified from entries where id = $1 and user_id = $2", this.Id, user_id)
		err := row.Scan(&e.RaffleId, &e.UserId, &e.Name, &e.Entered, &e.Disqualified)
		return err
	})
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return &e, nil
	}
}

func (this *Raffle) Enter(user_id int, display string) (bool, error) {
	var changed bool
	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		var entered bool
		row := tx.QueryRow("select entered from entries where id = $1 and user_id = $2", this.Id, user_id)
		err := row.Scan(&entered)
		if err != nil && err != sql.ErrNoRows {
			return err
		} else if entered {
			return nil
		}
		changed = true
		_, err = tx.Exec("insert into entries (id, user_id, display, entered) values ($1, $2, $3, $4) on conflict (id, user_id) do update set entered = $4", this.Id, user_id, display, true)
		return err
	})
	return changed, err
}

func (this *Raffle) Withdraw(user_id int, display string) (bool, error) {
	var changed bool
	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		var entered bool
		row := tx.QueryRow("select entered from entries where id = $1 and user_id = $2", this.Id, user_id)
		err := row.Scan(&entered)
		if err != nil && err != sql.ErrNoRows {
			return err
		} else if !entered {
			return nil
		}
		changed = true
		_, err = tx.Exec("insert into entries (id, user_id, display, entered) values ($1, $2, $3, $4) on conflict (id, user_id) do update set entered = $4", this.Id, user_id, display, false)
		return err
	})
	return changed, err
}

func (this *Raffle) Disqualify(user_id int) (bool, error) {
	var changed bool
	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		var disqualified bool
		row := tx.QueryRow("select disqualified from entries where id = $1 and user_id = $2", this.Id, user_id)
		err := row.Scan(&disqualified)
		if err != nil {
			return err
		} else if disqualified {
			return nil
		}
		changed = true
		_, err = tx.Exec("update entries set disqualified = $3 where id = $1 and user_id = $2", this.Id, user_id, true)
		return err
	})
	return changed, err
}

func (this *Raffle) Undisqualify(user_id int) (*Entry, error) {
	var e Entry
	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		_, err := tx.Exec("update entries set disqualified = $3 where id = $1 and user_id = $2", this.Id, user_id, false)
		if err != nil {
			return err
		}
		row := tx.QueryRow("select id, user_id, display, entered, disqualified from entries where id = $1 and user_id = $2", this.Id, user_id)
		return row.Scan(&e.RaffleId, &e.UserId, &e.Name, &e.Entered, &e.Disqualified)
	})
	if err != nil {
		return nil, err
	}
	return &e, nil
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

func Transact(object interface{}, parameters interface{}, db_func func(*sql.Tx, interface{}, interface{}) error) (error) {
	tx, err := connection.Begin()
	if err != nil {
		return err
	}

	err = db_func(tx, object, parameters)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}
