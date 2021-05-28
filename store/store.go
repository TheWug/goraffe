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
