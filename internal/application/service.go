package application

import (
	"crypto/rand"
	"encoding/hex"
	"oralarchive/internal/storage"
	"time"
)

type Service struct {
	store *storage.Store
	now   func() time.Time
}

func New(store *storage.Store) *Service {
	return &Service{store: store, now: time.Now}
}
func (s *Service) Store() *storage.Store { return s.store }

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
