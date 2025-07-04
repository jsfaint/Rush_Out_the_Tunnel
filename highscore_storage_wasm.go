//go:build js && wasm

package rush

import (
	"encoding/json"
	"syscall/js"
)

type wasmHighScoreStorage struct{}

func newHighScoreStorage() HighScoreStorage {
	return &wasmHighScoreStorage{}
}

const wasmHighScoreKey = "rush_highscores"

func (s *wasmHighScoreStorage) Save(highScores []HighScore) error {
	data, err := json.Marshal(highScores)
	if err != nil {
		return err
	}
	js.Global().Get("localStorage").Call("setItem", wasmHighScoreKey, string(data))
	return nil
}

func (s *wasmHighScoreStorage) Load() ([]HighScore, error) {
	item := js.Global().Get("localStorage").Call("getItem", wasmHighScoreKey)
	if item.IsNull() || item.IsUndefined() {
		return make([]HighScore, 5), nil
	}
	var loaded []HighScore
	if err := json.Unmarshal([]byte(item.String()), &loaded); err != nil {
		return nil, err
	}
	for len(loaded) < 5 {
		loaded = append(loaded, HighScore{"", 0})
	}
	return loaded[:5], nil
}
