package application

import (
	"crypto/rand"
	"encoding/hex"
	"oralarchive/internal/storage"
	"sync"
	"time"
)

type Service struct {
	store             *storage.Store
	now               func() time.Time
	verificationMu    sync.RWMutex
	verificationCache map[string][]byte
}

func New(store *storage.Store) *Service {
	return &Service{store: store, now: time.Now, verificationCache: make(map[string][]byte)}
}
func (s *Service) Store() *storage.Store { return s.store }

func (s *Service) cachedSuccessfulVerification(packageID string) ([]byte, bool) {
	s.verificationMu.RLock()
	defer s.verificationMu.RUnlock()
	data, ok := s.verificationCache[packageID]
	return append([]byte(nil), data...), ok
}

func (s *Service) cacheSuccessfulVerification(packageID string, data []byte) {
	s.verificationMu.Lock()
	defer s.verificationMu.Unlock()
	s.verificationCache[packageID] = append([]byte(nil), data...)
}

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
