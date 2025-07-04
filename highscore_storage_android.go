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

var customHighScoreDir string

// SetHighScoreDir 由 mobile 包调用，设置高分存储目录
func SetHighScoreDir(path string) {
	customHighScoreDir = path
}

func newHighScoreStorage() HighScoreStorage {
	// 优先使用 Java 层传递的目录
	dir := customHighScoreDir
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			dir = "."
		}
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
