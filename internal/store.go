package internal

import (
	"fmt"
	"sync"
	"time"
)

type Store struct {
	data      map[string]string
	expiry    map[string]int64 // key -> Unix timestamp
	mu        sync.RWMutex
	aof       *AOF
	replaying bool
}

func NewStore(aof *AOF) *Store {
	s := &Store{
		data:   make(map[string]string),
		expiry: make(map[string]int64),
		aof:    aof,
	}

	// Replay AOF
	if aof != nil {
		s.replaying = true
		aof.Replay(s)
		s.replaying = false
	}

	// Start TTL cleanup goroutine
	go s.cleanupExpiredKeys()
	return s
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	delete(s.expiry, key) // clear expiration on new set

	if s.aof != nil && !s.replaying {
		s.aof.AppendCommand("SET", key, value)
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check expiration
	if exp, ok := s.expiry[key]; ok {
		if time.Now().Unix() > exp {
			// Key expired
			s.mu.RUnlock()
			s.mu.Lock()
			delete(s.data, key)
			delete(s.expiry, key)
			if s.aof != nil && !s.replaying {
				s.aof.AppendCommand("DEL", key)
			}
			s.mu.Unlock()
			s.mu.RLock()
			return "", false
		}
	}

	val, ok := s.data[key]
	return val, ok
}

func (s *Store) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	if s.aof != nil {
		s.aof.AppendCommand("DEL", key)
	}
}

func (s *Store) Expire(key string, seconds int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[key]; !ok {
		return false
	}

	s.expiry[key] = time.Now().Unix() + seconds

	if s.aof != nil && !s.replaying {
		s.aof.AppendCommand("EXPIRE", key, fmt.Sprint(seconds))
	}
	return true
}

func (s *Store) cleanupExpiredKeys() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()
		s.mu.Lock()
		for key, exp := range s.expiry {
			if now > exp {
				delete(s.data, key)
				delete(s.expiry, key)
				if s.aof != nil && !s.replaying {
					s.aof.AppendCommand("DEL", key)
				}
			}
		}
		s.mu.Unlock()
	}
}
