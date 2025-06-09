package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Card represents a playing card.
type Card struct {
	ID   int
	Suit string
	Rank interface{} // Can be int or string ("J", "Q", "K", "A")
}

// Suits and Ranks for creating cards.
var Suits = []string{"♥", "♠", "♦", "♣"}
var Ranks = []interface{}{2, 3, 4, 5, 6, 7, 8, 9, 10, "J", "Q", "K", "A"}

// NewCard creates a new Card instance.
func NewCard(id int) Card {
	return Card{
		ID:   id,
		Suit: Suits[id/13],
		Rank: Ranks[id%13],
	}
}

// GetValue returns the numerical value of the card.
func (c *Card) GetValue() int {
	switch v := c.Rank.(type) {
	case string:
		if v == "J" || v == "Q" || v == "K" {
			return 10
		} else if v == "A" {
			return 11
		}
	case int:
		return v
	}
	return 0 // Should never happen
}

// Show returns a string representation of the card (e.g., "A♥").
func (c *Card) Show() string {
	return fmt.Sprintf("%v%s", c.Rank, c.Suit)
}

// Hand represents a player's hand of cards.
type Hand struct {
	Cards       []Card
	IsSplited   bool
	Status      string
	value       int
	aces        int
	isBusted    bool
	isBlackjack bool
}

// NewHand creates a new Hand instance.
func NewHand(cards []Card) *Hand {
	h := &Hand{
		Cards:     cards,
		IsSplited: false,
		Status:    "",
	}
	h.calculateValue()
	return h
}

// HitMe adds a card to the hand.
func (h *Hand) HitMe(card Card) {
	h.Cards = append(h.Cards, card)
	h.calculateValue()

}

// GetValue returns the total value of the hand.
func (h *Hand) GetValue() int {
	return h.value
}

func (h *Hand) calculateValue() {
	h.value = 0
	h.aces = 0
	for _, card := range h.Cards {
		if card.Rank == "A" {
			h.aces++
		}
		h.value += card.GetValue()
	}
	for h.value > 21 && h.aces > 0 {
		h.value -= 10
		h.aces--
	}
	h.isBusted = h.value > 21
	h.isBlackjack = len(h.Cards) == 2 && h.value == 21 && !h.IsSplited
}

// Show returns a string representation of the hand.
func (h *Hand) Show() string {
	cardStrings := make([]string, len(h.Cards))
	for i, card := range h.Cards {
		cardStrings[i] = card.Show()
	}
	return strings.Join(cardStrings, "  ")
}

// IsBlackjack returns true if the hand is a blackjack.
func (h *Hand) IsBlackjack() bool {
	return h.isBlackjack
}

// CanSplit returns true if the hand can be split.
func (h *Hand) CanSplit() bool {
	if len(h.Cards) != 2 || h.IsSplited {
		return false
	}
	return h.Cards[0].Rank == h.Cards[1].Rank
}

// IsBusted returns true if the hand is busted.
func (h *Hand) IsBusted() bool {
	return h.isBusted
}

// Deck represents a deck of cards.
type Deck struct {
	Cards []Card
}

// NewDeck creates a new Deck instance and shuffles it.
func NewDeck() *Deck {
	cards := make([]Card, 52)
	for i := 0; i < 52; i++ {
		cards[i] = NewCard(i)
	}

	// Shuffle the deck using Fisher-Yates algorithm
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})

	return &Deck{Cards: cards}
}

// Deal removes and returns the top card from the deck.
func (d *Deck) Deal() Card {
	card := d.Cards[0]
	d.Cards = d.Cards[1:]
	return card
}

// DealHand deals a hand of two cards.
func (d *Deck) DealHand() []Card {
	return []Card{d.Deal(), d.Deal()}
}

// Player represents a player in the game.
type Player struct {
	ID          int64
	Hands       []*Hand
	CurrentHand *Hand
}

// NewPlayer creates a new Player instance.
func NewPlayer(id int64) *Player {
	return &Player{
		ID:    id,
		Hands: []*Hand{},
	}
}

// Game represents a game of Blackjack.
type Game struct {
	Players map[int64]*Player
	Deck    *Deck
	Dealer  *Player
}

// NewGame creates a new Game instance.
func NewGame(players []*Player) *Game {
	playerMap := make(map[int64]*Player)
	for _, player := range players {
		playerMap[player.ID] = player
	}

	return &Game{
		Players: playerMap,
		Deck:    NewDeck(),
		Dealer:  NewPlayer(0), // Dealer's ID is 0
	}
}

// Start starts the game, dealing initial hands.
func (g *Game) Start() {
	g.Dealer.Hands = []*Hand{NewHand(g.Deck.DealHand())}
	g.Dealer.CurrentHand = g.Dealer.Hands[0]

	for _, player := range g.Players {
		player.Hands = []*Hand{NewHand(g.Deck.DealHand())}
		player.CurrentHand = player.Hands[0]
	}
}

// DealerTurn executes the dealer's turn.
func (g *Game) DealerTurn() {
	for g.Dealer.CurrentHand.GetValue() < 17 {
		g.Dealer.CurrentHand.HitMe(g.Deck.Deal())
	}
}

// Split splits the player's hand into two hands.
func (g *Game) Split(player *Player) {
	newHand := NewHand([]Card{player.Hands[0].Cards[1]})
	player.Hands[0].Cards = []Card{player.Hands[0].Cards[0]}
	player.Hands = append(player.Hands, newHand)

	player.Hands[0].HitMe(g.Deck.Deal())
	player.Hands[0].IsSplited = true
	player.Hands[1].HitMe(g.Deck.Deal())
	player.Hands[1].IsSplited = true

	player.CurrentHand = player.Hands[0]
}
