package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func GetGames(url string) (Games, error) {
	res, err := http.Get(url)
	if err != nil {
		return Games{}, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Games{}, err
	}
	res.Body.Close()

	games := Games{}
	json.Unmarshal(body, &games)

	return games, nil
}

func GameResult(game Game, player_id string) string {
	if game.White.Result != "win" && game.Black.Result != "win" {
		return "draw"
	}
	if game.White.Id == player_id && game.White.Result == "win" {
		return "win"
	}
	if game.Black.Id == player_id && game.Black.Result == "win" {
		return "win"
	}
	return "loss"
}

func PlayerCountryApi(player string) (string, error) {
	member_url := fmt.Sprintf("https://api.chess.com/pub/player/%s", player)
	b, err := http.Get(member_url)
	if err != nil {
		return "", err
	}
	bd, err := io.ReadAll(b.Body)
	if err != nil {
		return "", err
	}
	b.Body.Close()

	member := Member{}
	err = json.Unmarshal(bd, &member)
	if err != nil {
		return "", err
	}
	country_url := strings.Split(member.Country, "/")
	country := country_url[len(country_url)-1]
	return MapCountry(country), nil
}

func MapCountry(country string) string {
	switch country {
	case "XB", "XK", "XG":
		return "ES"
	case "XE", "XS", "XW":
		return "GB"
	default:
		return country
	}
}

type Member struct {
	Country string `json:"country"`
}

type Player struct {
	Result string `json:"result"`
	Id     string `json:"@id"`
}
type Game struct {
	White Player `json:"white"`
	Black Player `json:"black"`
	Rated bool   `json:"rated"`
}
type Games struct {
	Games []Game `json:"games"`
}
