package rush

type HighScoreStorage interface {
	Save(highScores []HighScore) error
	Load() ([]HighScore, error)
}

// NewHighScoreStorage 返回当前平台的高分存储实现
func NewHighScoreStorage() HighScoreStorage {
	// 具体实现由各平台的 build tag 文件提供
	return newHighScoreStorage()
}
