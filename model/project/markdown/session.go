package markdown

type Session struct {
	processedAssets map[string]bool
}

func NewSession() *Session {
	return &Session{processedAssets: make(map[string]bool)}
}
