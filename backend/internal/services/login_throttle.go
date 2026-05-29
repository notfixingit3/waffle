package services

import (
	"fmt"
	"sync"
	"time"
)

const (
	maxFailedAttempts = 5
	lockoutDuration   = 15 * time.Minute
	cleanupInterval   = 5 * time.Minute
)

type throttleEntry struct {
	attempts   int
	lockedAt   *time.Time
	lastFailed time.Time
}

type LoginThrottle struct {
	mu      sync.Mutex
	entries map[string]*throttleEntry
}

var loginThrottle = &LoginThrottle{
	entries: make(map[string]*throttleEntry),
}

func init() {
	go func() {
		for {
			time.Sleep(cleanupInterval)
			loginThrottle.cleanup()
		}
	}()
}

func makeKey(ip, username string) string {
	return fmt.Sprintf("%s|%s", ip, username)
}

func (lt *LoginThrottle) RecordFailedAttempt(ip, username string) bool {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := makeKey(ip, username)
	entry, exists := lt.entries[key]
	if !exists {
		entry = &throttleEntry{}
		lt.entries[key] = entry
	}

	entry.attempts++
	entry.lastFailed = time.Now()

	if entry.attempts >= maxFailedAttempts {
		now := time.Now()
		entry.lockedAt = &now
		return true
	}
	return false
}

func (lt *LoginThrottle) IsLockedOut(ip, username string) bool {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := makeKey(ip, username)
	entry, exists := lt.entries[key]
	if !exists {
		return false
	}

	if entry.lockedAt == nil {
		return false
	}

	if time.Since(*entry.lockedAt) >= lockoutDuration {
		// Lock has expired — clear it
		delete(lt.entries, key)
		return false
	}

	return true
}

func (lt *LoginThrottle) ResetAttempts(ip, username string) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	key := makeKey(ip, username)
	delete(lt.entries, key)
}

func (lt *LoginThrottle) cleanup() {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	now := time.Now()
	for key, entry := range lt.entries {
		if entry.lockedAt != nil && now.Sub(*entry.lockedAt) >= lockoutDuration {
			delete(lt.entries, key)
			continue
		}
		// Also clean up stale entries that haven't seen activity in a while
		if now.Sub(entry.lastFailed) >= lockoutDuration {
			delete(lt.entries, key)
		}
	}
}

// Package-level functions for convenience
func RecordFailedLoginAttempt(ip, username string) bool {
	return loginThrottle.RecordFailedAttempt(ip, username)
}

func IsLoginLockedOut(ip, username string) bool {
	return loginThrottle.IsLockedOut(ip, username)
}

func ResetLoginAttempts(ip, username string) {
	loginThrottle.ResetAttempts(ip, username)
}
