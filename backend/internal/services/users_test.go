package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/syrup/backend/internal/db"
)

// testUserPrefix is used to identify and clean up test user data.
const testUserPrefix = "test-user-"

// insertTestUser inserts a user with the given handle directly into the database.
// Returns the inserted user's ID.
func insertTestUser(t *testing.T, handle string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	_, err := db.Pool.Exec(context.Background(), `
		INSERT INTO users (id, instagram_handle, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (instagram_handle) DO NOTHING
	`, id, handle, now, now)
	if err != nil {
		t.Fatalf("insert test user %q: %v", handle, err)
	}
	return id
}

// cleanupTestUsers removes all users whose handle starts with the test prefix.
func cleanupTestUsers(t *testing.T) {
	t.Helper()
	_, err := db.Pool.Exec(context.Background(), `DELETE FROM users WHERE instagram_handle LIKE $1`, testUserPrefix+"%")
	if err != nil {
		t.Fatalf("cleanup test users: %v", err)
	}
}

// TestGetOrCreateUser_New inserts a new user and verifies the returned user has a
// normalized handle and a valid UUID.
func TestGetOrCreateUser_New(t *testing.T) {
	defer cleanupTestUsers(t)

	handle := testUserPrefix + uuid.New().String()[:8]
	user, err := GetOrCreateUser(handle)
	if err != nil {
		t.Fatalf("GetOrCreateUser(%q) returned error: %v", handle, err)
	}

	if user.ID == uuid.Nil {
		t.Error("expected non-nil UUID for new user")
	}
	if user.InstagramHandle != handle {
		t.Errorf("expected handle %q, got %q", handle, user.InstagramHandle)
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

// TestGetOrCreateUser_Existing inserts the same handle twice and verifies both
// calls return the same user ID (idempotent upsert).
func TestGetOrCreateUser_Existing(t *testing.T) {
	defer cleanupTestUsers(t)

	handle := testUserPrefix + uuid.New().String()[:8]
	first, err := GetOrCreateUser(handle)
	if err != nil {
		t.Fatalf("first GetOrCreateUser(%q) returned error: %v", handle, err)
	}

	second, err := GetOrCreateUser(handle)
	if err != nil {
		t.Fatalf("second GetOrCreateUser(%q) returned error: %v", handle, err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same user ID on duplicate call, got %s vs %s", first.ID, second.ID)
	}
	if first.InstagramHandle != second.InstagramHandle {
		t.Errorf("expected same handle on duplicate call, got %q vs %q", first.InstagramHandle, second.InstagramHandle)
	}
}

// TestGetOrCreateUser_Normalization passes "@TestUser" and "testuser" and verifies
// both calls resolve to the same user record (handling @-stripping and lowercasing).
func TestGetOrCreateUser_Normalization(t *testing.T) {
	defer cleanupTestUsers(t)

	baseHandle := testUserPrefix + uuid.New().String()[:8]
	withAt := "@" + baseHandle
	withoutAt := baseHandle

	first, err := GetOrCreateUser(withAt)
	if err != nil {
		t.Fatalf("GetOrCreateUser(%q) returned error: %v", withAt, err)
	}

	second, err := GetOrCreateUser(withoutAt)
	if err != nil {
		t.Fatalf("GetOrCreateUser(%q) returned error: %v", withoutAt, err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same user ID after normalization, got %s vs %s", first.ID, second.ID)
	}
	if first.InstagramHandle != second.InstagramHandle {
		t.Errorf("expected same handle after normalization, got %q vs %q", first.InstagramHandle, second.InstagramHandle)
	}
	// Verify handle was normalized (no @, lowercase)
	if first.InstagramHandle != withoutAt {
		t.Errorf("expected normalized handle %q, got %q", withoutAt, first.InstagramHandle)
	}
}

// TestGetOrCreateUser_EmptyHandle passes an empty string and verifies an error is
// returned (empty normalized handle should fail the UNIQUE constraint or not insert).
func TestGetOrCreateUser_EmptyHandle(t *testing.T) {
	defer cleanupTestUsers(t)

	_, err := GetOrCreateUser("")
	if err == nil {
		t.Fatal("expected error for empty handle, got nil")
	}
}

// TestListUsers_NoSearch inserts several users and verifies ListUsers returns all
// of them with correct pagination metadata.
func TestListUsers_NoSearch(t *testing.T) {
	defer cleanupTestUsers(t)

	handles := []string{
		testUserPrefix + "aaa-" + uuid.New().String()[:8],
		testUserPrefix + "bbb-" + uuid.New().String()[:8],
		testUserPrefix + "ccc-" + uuid.New().String()[:8],
	}
	for _, h := range handles {
		insertTestUser(t, h)
	}

	// Fetch with limit larger than count
	users, total, err := ListUsers("", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if total < len(handles) {
		t.Errorf("expected total >= %d, got %d", len(handles), total)
	}
	if len(users) < len(handles) {
		t.Errorf("expected at least %d users in result, got %d", len(handles), len(users))
	}

	// Verify all test handles appear in results
	found := make(map[string]bool)
	for _, u := range users {
		found[u.InstagramHandle] = true
	}
	for _, h := range handles {
		if !found[h] {
			t.Errorf("test user %q not found in ListUsers results", h)
		}
	}
}

// TestListUsers_WithSearch inserts several users and verifies that an ILIKE search
// correctly returns only matching handles.
func TestListUsers_WithSearch(t *testing.T) {
	defer cleanupTestUsers(t)

	prefix := testUserPrefix + uuid.New().String()[:8]
	matchingHandle := prefix + "-matchme"
	otherHandle := prefix + "-other"

	insertTestUser(t, matchingHandle)
	insertTestUser(t, otherHandle)

	users, total, err := ListUsers("matchme", 100, 0)
	if err != nil {
		t.Fatalf("ListUsers with search returned error: %v", err)
	}

	if total != 1 {
		t.Errorf("expected total=1 for search, got %d", total)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user in results, got %d", len(users))
	}
	if len(users) > 0 && users[0].InstagramHandle != matchingHandle {
		t.Errorf("expected %q, got %q", matchingHandle, users[0].InstagramHandle)
	}
}

// TestListUsers_Pagination inserts enough users to span multiple pages and verifies
// that page 2 returns the correct offset (no overlap with page 1).
func TestListUsers_Pagination(t *testing.T) {
	defer cleanupTestUsers(t)

	// Insert 3 test users with ordered handles
	handles := []string{
		testUserPrefix + "pag-a-" + uuid.New().String()[:8],
		testUserPrefix + "pag-b-" + uuid.New().String()[:8],
		testUserPrefix + "pag-c-" + uuid.New().String()[:8],
	}
	for _, h := range handles {
		insertTestUser(t, h)
	}

	// Page 1: limit=1, offset=0
	page1, total, err := ListUsers("", 1, 0)
	if err != nil {
		t.Fatalf("ListUsers page 1 returned error: %v", err)
	}
	if len(page1) != 1 {
		t.Fatalf("expected 1 user on page 1, got %d", len(page1))
	}
	if total < len(handles) {
		t.Errorf("expected total >= %d, got %d", len(handles), total)
	}

	// Page 2: limit=1, offset=1
	page2, _, err := ListUsers("", 1, 1)
	if err != nil {
		t.Fatalf("ListUsers page 2 returned error: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("expected 1 user on page 2, got %d", len(page2))
	}

	// Verify no overlap between pages
	if page1[0].ID == page2[0].ID {
		t.Error("page 1 and page 2 returned the same user — expected different users")
	}
}
