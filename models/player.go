package models

import (
	"encoding/json"
	"strings"

	"github.com/guregu/null"
)

// Player struct used for storing players
type Player struct {
	ID             int         `json:"id"`
	FirstName      string      `json:"first_name"`
	LastName       null.String `json:"last_name"`
	VocalName      null.String `json:"vocal_name,omitempty"`
	Nickname       null.String `json:"nickname,omitempty"`
	SlackHandle    null.String `json:"slack_handle,omitempty"`
	MatchesPlayed  int         `json:"matches_played"`
	MatchesWon     int         `json:"matches_won"`
	LegsPlayed     int         `json:"legs_played"`
	LegsWon        int         `json:"legs_won"`
	Color          null.String `json:"color,omitempty"`
	ProfilePicURL  null.String `json:"profile_pic_url,omitempty"`
	SmartcardUID   null.String `json:"smartcard_uid,omitempty"`
	BoardStreamURL null.String `json:"board_stream_url,omitempty"`
	BoardStreamCSS null.String `json:"board_stream_css,omitempty"`
	OfficeID       null.Int    `json:"office_id,omitempty"`
	IsActive       bool        `json:"is_active"`
	IsBot          bool        `json:"is_bot"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at,omitempty"`
	TournamentElo  int         `json:"tournament_elo,omitempty"`
	CurrentElo     int         `json:"current_elo,omitempty"`
}

// PlayerStatistics used to store player statistics
type PlayerStatistics struct {
	X01      *StatisticsX01      `json:"x01"`
	Shootout *StatisticsShootout `json:"shootout"`
	Cricket  *StatisticsCricket  `json:"cricket"`
	DartsAt  *StatisticsDartsAtX `json:"darts_at_x"`
}

// MarshalJSON will marshall the given object to JSON
func (player Player) MarshalJSON() ([]byte, error) {
	// Use a type to get consistnt order of JSON key-value pairs.
	type playerJSON struct {
		ID             int         `json:"id"`
		Name           string      `json:"name"`
		FirstName      string      `json:"first_name"`
		LastName       null.String `json:"last_name"`
		VocalName      null.String `json:"vocal_name,omitempty"`
		Nickname       null.String `json:"nickname,omitempty"`
		SlackHandle    null.String `json:"slack_handle,omitempty"`
		MatchesPlayed  int         `json:"matches_played"`
		MatchesWon     int         `json:"matches_won"`
		LegsPlayed     int         `json:"legs_played"`
		LegsWon        int         `json:"legs_won"`
		Color          null.String `json:"color,omitempty"`
		ProfilePicURL  null.String `json:"profile_pic_url,omitempty"`
		SmartcardUID   null.String `json:"smartcard_uid,omitempty"`
		BoardStreamURL null.String `json:"board_stream_url,omitempty"`
		BoardStreamCSS null.String `json:"board_stream_css,omitempty"`
		OfficeID       null.Int    `json:"office_id,omitempty"`
		IsActive       bool        `json:"is_active"`
		IsBot          bool        `json:"is_bot"`
		CreatedAt      string      `json:"created_at"`
		UpdatedAt      string      `json:"updated_at,omitempty"`
		TournamentElo  int         `json:"tournament_elo,omitempty"`
		CurrentElo     int         `json:"current_elo,omitempty"`
	}

	return json.Marshal(playerJSON{
		ID:             player.ID,
		FirstName:      player.FirstName,
		LastName:       player.LastName,
		VocalName:      player.VocalName,
		Nickname:       player.Nickname,
		SlackHandle:    player.SlackHandle,
		MatchesPlayed:  player.MatchesPlayed,
		MatchesWon:     player.MatchesWon,
		LegsPlayed:     player.LegsPlayed,
		LegsWon:        player.LegsWon,
		Color:          player.Color,
		ProfilePicURL:  player.ProfilePicURL,
		SmartcardUID:   player.SmartcardUID,
		BoardStreamURL: player.BoardStreamURL,
		BoardStreamCSS: player.BoardStreamCSS,
		OfficeID:       player.OfficeID,
		IsActive:       player.IsActive,
		IsBot:          player.IsBot,
		CreatedAt:      player.CreatedAt,
		UpdatedAt:      player.UpdatedAt,
		TournamentElo:  player.TournamentElo,
		CurrentElo:     player.CurrentElo,
		Name:           strings.Trim((player.FirstName + " " + player.LastName.ValueOrZero()), " "),
	})
}
