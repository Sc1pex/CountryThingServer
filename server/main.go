package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

func main() {
	http.HandleFunc("/{user}", ws)
	http.ListenAndServe(":3000", nil)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func ws(w http.ResponseWriter, r *http.Request) {
	user := r.PathValue("user")
	fmt.Println("User:", user)

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer ws.Close()

	api_mutex := sync.Mutex{}
	user_id := fmt.Sprintf("https://api.chess.com/pub/player/%s", user)
	games_url := fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", user)

	res, err := http.Get(games_url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	archives := Archives{}
	err = json.Unmarshal(body, &archives)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	p_per_archive := 100.0 / float64(len(archives.Archives))
	p := 0.0

	for _, archive := range archives.Archives {
		fmt.Printf("Getting games from url: %s\n", archive)
		games, err := GetGames(archive)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		result := make(chan ProcessResult, len(games.Games))
		var wg sync.WaitGroup

		for _, game := range games.Games {
			wg.Add(1)

			go processGame(game, user_id, &wg, &api_mutex, result)
		}

		go func() {
			wg.Wait()
			close(result)
		}()

		p_per_game := p_per_archive / float64(len(games.Games))
		for res := range result {
			if res.Error != nil {
				fmt.Println("Error:", res.Error)
				return
			}
			p += p_per_game
			ws.WriteMessage(1, []byte(fmt.Sprintf("%s %s %.3f", res.Result.Country, res.Result.Score, p)))
		}
	}
}

type CountryEntry struct {
	Wins   int
	Losses int
	Draws  int
}

type ProcessResult struct {
	Result struct {
		Country string
		Score   string
	}
	Error error
}

func processGame(
	game Game,
	user_id string,
	wg *sync.WaitGroup,
	api_mutex *sync.Mutex,
	ch chan ProcessResult,
) {
	defer wg.Done()

	if !game.Rated {
		return
	}

	opponent := game.White.Id
	if game.White.Id == user_id {
		opponent = game.Black.Id
	}
	opponent_s := strings.Split(opponent, "/")
	opponent = opponent_s[len(opponent_s)-1]

	api_mutex.Lock()
	opponent_country, err := PlayerCountryApi(opponent)
	api_mutex.Unlock()

	if err != nil {
		fmt.Println("Error:", err)
		ch <- ProcessResult{Error: err}
		return
	}

	result := GameResult(game, user_id)
	ch <- ProcessResult{
		Error: nil,
		Result: struct {
			Country string
			Score   string
		}{
			Country: opponent_country,
			Score:   result,
		},
	}
}

type Archives struct {
	Archives []string `json:"archives"`
}
