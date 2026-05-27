package services

import (
	"testing"
	"time"
)

func TestLoginThrottle_RecordFailedAttempt_LocksAfterFive(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	ip := "203.0.113.1"
	username := "admin"

	// 4 failures should not lock
	for i := 0; i < 4; i++ {
		locked := lt.RecordFailedAttempt(ip, username)
		if locked {
			t.Fatalf("expected not locked on attempt %d", i+1)
		}
	}

	if lt.IsLockedOut(ip, username) {
		t.Fatal("expected not locked out after 4 attempts")
	}

	// 5th failure should trigger lock
	locked := lt.RecordFailedAttempt(ip, username)
	if !locked {
		t.Fatal("expected locked after 5th attempt")
	}

	if !lt.IsLockedOut(ip, username) {
		t.Fatal("expected locked out after 5 attempts")
	}
}

func TestLoginThrottle_IsLockedOut_Expires(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	ip := "203.0.113.2"
	username := "admin"

	// Trigger lockout
	for i := 0; i < 5; i++ {
		lt.RecordFailedAttempt(ip, username)
	}

	if !lt.IsLockedOut(ip, username) {
		t.Fatal("expected locked out immediately")
	}

	// Manually backdate the lock to simulate expiry
	lt.mu.Lock()
	key := makeKey(ip, username)
	entry := lt.entries[key]
	expired := time.Now().Add(-16 * time.Minute)
	entry.lockedAt = &expired
	lt.mu.Unlock()

	if lt.IsLockedOut(ip, username) {
		t.Fatal("expected not locked out after expiry")
	}

	// Entry should be cleaned up after expiry check
	lt.mu.Lock()
	_, exists := lt.entries[key]
	lt.mu.Unlock()
	if exists {
		t.Error("expected expired entry to be removed")
	}
}

func TestLoginThrottle_ResetAttempts(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	ip := "203.0.113.3"
	username := "admin"

	// Trigger lockout
	for i := 0; i < 5; i++ {
		lt.RecordFailedAttempt(ip, username)
	}

	if !lt.IsLockedOut(ip, username) {
		t.Fatal("expected locked out before reset")
	}

	lt.ResetAttempts(ip, username)

	if lt.IsLockedOut(ip, username) {
		t.Fatal("expected not locked out after reset")
	}

	// Verify entry is gone
	lt.mu.Lock()
	_, exists := lt.entries[makeKey(ip, username)]
	lt.mu.Unlock()
	if exists {
		t.Error("expected entry to be removed after reset")
	}
}

func TestLoginThrottle_PerIPIsolation(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	username := "admin"
	ip1 := "203.0.113.10"
	ip2 := "203.0.113.20"

	// Lock out ip1
	for i := 0; i < 5; i++ {
		lt.RecordFailedAttempt(ip1, username)
	}

	if !lt.IsLockedOut(ip1, username) {
		t.Fatal("expected ip1 to be locked out")
	}

	// ip2 should not be affected
	if lt.IsLockedOut(ip2, username) {
		t.Fatal("expected ip2 to NOT be locked out")
	}

	// ip2 should still be able to fail without immediate lock
	locked := lt.RecordFailedAttempt(ip2, username)
	if locked {
		t.Fatal("expected ip2 first attempt not to lock")
	}
}

func TestLoginThrottle_PerUsernameIsolation(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	ip := "203.0.113.30"
	user1 := "admin"
	user2 := "other"

	// Lock out user1
	for i := 0; i < 5; i++ {
		lt.RecordFailedAttempt(ip, user1)
	}

	if !lt.IsLockedOut(ip, user1) {
		t.Fatal("expected user1 to be locked out")
	}

	// user2 from same IP should not be affected
	if lt.IsLockedOut(ip, user2) {
		t.Fatal("expected user2 to NOT be locked out")
	}
}

func TestLoginThrottle_CleanupRemovesExpired(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	ip := "203.0.113.4"
	username := "admin"

	// Trigger lockout
	for i := 0; i < 5; i++ {
		lt.RecordFailedAttempt(ip, username)
	}

	// Backdate to simulate expiry
	lt.mu.Lock()
	key := makeKey(ip, username)
	entry := lt.entries[key]
	expired := time.Now().Add(-16 * time.Minute)
	entry.lockedAt = &expired
	lt.mu.Unlock()

	// Run cleanup
	lt.cleanup()

	lt.mu.Lock()
	_, exists := lt.entries[key]
	lt.mu.Unlock()
	if exists {
		t.Error("expected cleanup to remove expired entry")
	}
}

func TestLoginThrottle_CleanupRemovesStaleUnlocked(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	ip := "203.0.113.5"
	username := "admin"

	// Just 1 failed attempt (not locked)
	lt.RecordFailedAttempt(ip, username)

	// Backdate lastFailed to simulate staleness
	lt.mu.Lock()
	key := makeKey(ip, username)
	entry := lt.entries[key]
	entry.lastFailed = time.Now().Add(-16 * time.Minute)
	lt.mu.Unlock()

	// Run cleanup
	lt.cleanup()

	lt.mu.Lock()
	_, exists := lt.entries[key]
	lt.mu.Unlock()
	if exists {
		t.Error("expected cleanup to remove stale unlocked entry")
	}
}

func TestLoginThrottle_PackageFunctions(t *testing.T) {
	// Reset global state for this test
	loginThrottle.mu.Lock()
	loginThrottle.entries = make(map[string]*throttleEntry)
	loginThrottle.mu.Unlock()

	ip := "203.0.113.6"
	username := "admin"

	// Use package-level functions
	for i := 0; i < 4; i++ {
		locked := RecordFailedLoginAttempt(ip, username)
		if locked {
			t.Fatalf("expected not locked on attempt %d", i+1)
		}
	}

	locked := RecordFailedLoginAttempt(ip, username)
	if !locked {
		t.Fatal("expected locked after 5th attempt")
	}

	if !IsLoginLockedOut(ip, username) {
		t.Fatal("expected locked out via package function")
	}

	ResetLoginAttempts(ip, username)

	if IsLoginLockedOut(ip, username) {
		t.Fatal("expected not locked out after reset via package function")
	}
}

func TestLoginThrottle_IsLockedOut_NonExistent(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	if lt.IsLockedOut("1.2.3.4", "nobody") {
		t.Fatal("expected non-existent entry to not be locked out")
	}
}

func TestLoginThrottle_RecordFailedAttempt_ReturnsTrueOnlyOnThreshold(t *testing.T) {
	lt := &LoginThrottle{
		entries: make(map[string]*throttleEntry),
	}

	ip := "203.0.113.7"
	username := "admin"

	// Attempts 1-4 should return false
	for i := 0; i < 4; i++ {
		locked := lt.RecordFailedAttempt(ip, username)
		if locked {
			t.Fatalf("attempt %d: expected locked=false, got true", i+1)
		}
	}

	// Attempt 5 should return true (threshold reached)
	locked := lt.RecordFailedAttempt(ip, username)
	if !locked {
		t.Fatal("attempt 5: expected locked=true, got false")
	}

	// Attempt 6+ should also return true (already locked)
	locked = lt.RecordFailedAttempt(ip, username)
	if !locked {
		t.Fatal("attempt 6: expected locked=true, got false")
	}
}
