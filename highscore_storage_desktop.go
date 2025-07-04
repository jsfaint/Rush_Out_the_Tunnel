//go:build !android && !js

package rush

import (
	"encoding/json"
	"os"
)

type desktopHighScoreStorage struct{}

func newHighScoreStorage() HighScoreStorage {
	return &desktopHighScoreStorage{}
}

const desktopHighScoreFile = "highscores.json"

func (s *desktopHighScoreStorage) Save(highScores []HighScore) error {
	file, err := os.Create(desktopHighScoreFile)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	return enc.Encode(highScores)
}

func (s *desktopHighScoreStorage) Load() ([]HighScore, error) {
	file, err := os.Open(desktopHighScoreFile)
	if err != nil {
		// 文件不存在时返回空榜
		return make([]HighScore, 5), nil
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	var loaded []HighScore
	if err := dec.Decode(&loaded); err != nil {
		return nil, err
	}
	// 保证长度为5
	for len(loaded) < 5 {
		loaded = append(loaded, HighScore{"", 0})
	}
	return loaded[:5], nil
}
