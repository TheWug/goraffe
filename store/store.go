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
	if this.IsOpen {
		return false, nil
	}

	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		_, err := tx.Exec("update raffles set open = $1 where id = $2", true, this.Id)
		return err
	})

	if err != nil {
		return false, err
	}

	this.IsOpen = true
	return true, nil
}

func (this *Raffle) Close() (bool, error) {
	changed := this.IsOpen
	if !changed {
		return changed, nil
	}

	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		_, err := tx.Exec("update raffles set open = $1 where id = $2", false, this.Id)
		return err
	})

	if err != nil {
		return false, err
	}

	this.IsOpen = false
	return changed, nil
}

func (this *Raffle) Cancel() (bool, error) {
	return true, Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		_, err := tx.Exec("delete from entries where id = $1", this.Id)
		return err
	})
}

func (this *Raffle) Draw() (*Entry, error) {
	var winner *Entry
	err := Transact(nil, nil, func(tx *sql.Tx, a, b interface{}) error {
		this.IsOpen = false
		_, err := tx.Exec("update raffles set open = false where id = $1", this.Id)
		if err != nil {
			return err
		}

		var entries []Entry
		var scores []Score

		// get the current state of scores
		rows, err := tx.Query("select id, user_id, display from entries where id = $1 and entered = true and disqualified = false", this.Id)
		if err != nil {
			return err
		}
		for rows.Next() {
			e := Entry{Entered: true}
			err = rows.Scan(&e.RaffleId, &e.UserId, &e.Name)
			if err != nil {
				return err
			}
			entries = append(entries, e)
		}

		rows, err = tx.Query("select id, user_id, display, score, lifetime_score from scores where id = $1", this.Id)
		if err != nil {
			return err
		}
		for rows.Next() {
			s := Score{}
			err = rows.Scan(&s.RaffleId, &s.UserId, &s.Name, &s.Score, &s.LifetimeScore)
			if err != nil {
				return err
			}
			scores = append(scores, s)
		}

		// actually draw the raffle.
		winner, scores = RaffleDraw(this.Id, entries, scores)

		// delete all entries and store updated scores.
		_, err = tx.Exec("delete from entries where id = $1", this.Id)
		if err != nil {
			return err
		}

		for _, s := range scores {
			_, err = tx.Exec("insert into scores (id, user_id, display, score, lifetime_score) values ($1, $2, $3, $4, $5) on conflict (id, user_id) do update set score = $4, lifetime_score = $5", s.RaffleId, s.UserId, s.Name, s.Score, s.LifetimeScore)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return winner, nil
}

func CreateRaffle(owner int, name string, tiers []int32) (*Raffle, error) {
	tx, err := connection.Begin()
	if err != nil {
		return nil, err
	}

	u := uuid.New()
	t := time.Now()

	_, err = tx.Exec("insert into raffles (id, display, ts, owner, tiers) values ($1, $2, $3, $4, $5)",
	                  u.String(), name, t.Unix(), owner, pq.Array(tiers))

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	raffle := Raffle{
		Id: u.String(),
		Owner: owner,
		Tiers: tiers,
		Timestamp: t.Unix(),
	}

	return &raffle, nil
}

func GetRaffle(tx *sql.Tx, object interface{}, parameter interface{}) error {
	raffle_id, t1 := parameter.(string)
	raffle, t2 := object.(*Raffle)
	if !(t1 && t2) {
		return errors.New("Invalid parameters to GetRaffle")
	}

	row := tx.QueryRow("select id, display, ts, owner, tiers, open from raffles where id = $1", raffle_id)

	var tiers pq.Int32Array
	err := row.Scan(&raffle.Id, &raffle.Display, &raffle.Timestamp, &raffle.Owner, &tiers, &raffle.IsOpen)
	raffle.Tiers = []int32(tiers)
	if err == sql.ErrNoRows {
		err = nil // this is a non-error and simply means there was no matching raffle
	}

	return err
}

func GetRafflesFromList(tx *sql.Tx, object, parameter interface{}, query string) error {
	user_id, t1 := parameter.(int)
	raffles, t2 := object.(*[]Raffle)
	if !(t1 && t2) {
		return errors.New("Invalid parameters to GetRafflesFromList")
	}

	rows, err := tx.Query(query, user_id)
	if err != nil {
		return err
	}

	for rows.Next() {
		var raffle Raffle
		var tiers pq.Int32Array
		err = rows.Scan(&raffle.Id, &raffle.Display, &raffle.Timestamp, &raffle.Owner, &tiers, &raffle.IsOpen)
		if err != nil {
			*raffles = nil
			return err
		}
		raffle.Tiers = []int32(tiers)

		*raffles = append(*raffles, raffle)
	}

	return nil
}

func GetMyRaffles(tx *sql.Tx, object interface{}, parameter interface{}) error {
	return GetRafflesFromList(tx, object, parameter, "select id, display, ts, owner, tiers, open from raffles where owner = $1 order by ts asc")
}

func GetEnteredRaffles(tx *sql.Tx, object interface{}, parameter interface{}) error {
	return GetRafflesFromList(tx, object, parameter, "select raffles.id, raffles.display, raffles.ts, raffles.owner, raffles.tiers, raffles.open from raffles inner join entries using (id) where user_id = $1 order by ts asc")
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
