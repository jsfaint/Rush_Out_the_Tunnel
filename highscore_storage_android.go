//go:build android

package rush

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type androidHighScoreStorage struct {
	filePath string
}

func newHighScoreStorage() HighScoreStorage {
	// 默认存储到应用私有目录
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return &androidHighScoreStorage{
		filePath: filepath.Join(dir, "highscores.json"),
	}
}

func (s *androidHighScoreStorage) Save(highScores []HighScore) error {
	file, err := os.Create(s.filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	return enc.Encode(highScores)
}

func (s *androidHighScoreStorage) Load() ([]HighScore, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		return make([]HighScore, 5), nil
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	var loaded []HighScore
	if err := dec.Decode(&loaded); err != nil {
		return nil, err
	}
	for len(loaded) < 5 {
		loaded = append(loaded, HighScore{"", 0})
	}
	return loaded[:5], nil
}
