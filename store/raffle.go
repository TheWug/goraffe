package store

import (
	"math/rand"
)

// Choose who wins. This is the magic algorithm.
// In order to be compatible with patreons rules, raffles are not chosen randomly.
// They are instead chosen deterministically, by a process which produces the exact
// same win distribution as a fair raffle run under the same circumstances would.
// 
// In essence, the raffle is a game which is played simply by participating.
// Each round, one "probability point" is divided evenly amongst all entered players,
// and the player with the highest accumulated probability score wins the round, and loses
// one point from their total score.  Because 1 point is added and deducted to the sum
// of all players each round, the net score is always zero.
//
// Because scores can ONLY increase by playing unless you win a round, and can only
// decrease IF you win a round, it is mathematically provable that no matter what the
// initial conditions are, if you enter enough rounds, you will eventually reach a round
// where your probability of winning is 100%.
//
// The only exception is if a tie occurs: ties are broken by randomly selecting an
// entrant. If you have a population of 25 players, and you run one game, a winner will
// be selected randomly. But, if you run 24 more games, and each player enters every
// game, each player will win exactly one time (though in a random order). This is
// guaranteed, because a player who wins the drawing will lose a point, and no longer be
// considered when breaking ties until all other players have won at least once.
//
// The expected number of wins, given real world conditions (in particular, varying
// numbers of entrants per round) is exactly the same as with a true, fair raffle.


// precondition: entries must be filtered to include only active, non-disqualified ones
func RaffleDraw(raffle_id string, entries []Entry, scores []Score) (*Entry, []Score) {
	scoremap := make(map[int]int, len(scores))

	for i, _ := range scores {
		scoremap[scores[i].UserId] = i
	}
	
	// behold, the almighty point.
	var point float64 = 1.0
	points_per_entrant := point / float64(len(entries))
	
	var candidates []*Entry
	
	for i, _ := range entries {
		// look up the user's existing score
		n, ok := scoremap[entries[i].UserId]
		if !ok {
			// if the user has no score, start them off at zero
			n = len(scores)
			scoremap[entries[i].UserId] = n
			scores = append(scores, Score{
				RaffleId: raffle_id,
				UserId: entries[i].UserId,
				Name: entries[i].Name,
				Score: 0.0,
				LifetimeScore: 0.0,
			})
		}
		s := &scores[n]
		
		// the algorithm giveth!
		s.Score += points_per_entrant
		s.LifetimeScore += points_per_entrant
		
		// build a list containing all of the entries tied for the high score
		if len(candidates) == 0 {
			candidates = append(candidates, &entries[i])
		} else if x, ok := scoremap[candidates[0].UserId]; ok && scores[x].Score == s.Score {
			candidates = append(candidates, &entries[i])
		} else if x, ok := scoremap[candidates[0].UserId]; ok && scores[x].Score < s.Score {
			candidates = nil
			candidates = append(candidates, &entries[i])
		}
	}
	
	// behold, the chosen one
	winner := candidates[rand.Intn(len(candidates))]
	
	// and the algorithm taketh away.
	scores[scoremap[winner.UserId]].Score -= 1.0
	
	return winner, scores
}
