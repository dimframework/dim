package dim

import (
	"context"
	"testing"
)

func TestMockUserStoreCreate(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	err := store.Create(ctx, user)
	if err != nil {
		t.Errorf("Create() error = %v", err)
	}

	if user.ID != 1 {
		t.Errorf("user ID = %d, want 1", user.ID)
	}
}

func TestMockUserStoreFindByID(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)

	found, err := store.FindByID(ctx, user.ID)
	if err != nil {
		t.Errorf("FindByID() error = %v", err)
	}

	if found.Email != user.Email {
		t.Errorf("found email = %s, want %s", found.Email, user.Email)
	}
}

func TestMockUserStoreFindByEmail(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)

	found, err := store.FindByEmail(ctx, user.Email)
	if err != nil {
		t.Errorf("FindByEmail() error = %v", err)
	}

	if found.Email != user.Email {
		t.Errorf("found email = %s, want %s", found.Email, user.Email)
	}
}

func TestMockUserStoreUpdate(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)

	user.Email = "updated@example.com"
	err := store.Update(ctx, user)
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	found, _ := store.FindByID(ctx, user.ID)
	if found.Email != "updated@example.com" {
		t.Errorf("email not updated")
	}
}

func TestMockUserStoreDelete(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)

	err := store.Delete(ctx, user.ID)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	_, err = store.FindByID(ctx, user.ID)
	if err == nil {
		t.Errorf("user should not exist after deletion")
	}
}

func TestMockUserStoreExists(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)

	exists, err := store.Exists(ctx, user.Email)
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}

	if !exists {
		t.Errorf("user should exist")
	}

	exists, _ = store.Exists(ctx, "nonexistent@example.com")
	if exists {
		t.Errorf("nonexistent user should not exist")
	}
}

func TestMockUserStoreUpdatePartialEmail(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)
	userID := user.ID

	req := &UpdateUserRequest{
		Email: NewJsonNull("newemail@example.com"),
	}

	err := store.UpdatePartial(ctx, userID, req)
	if err != nil {
		t.Errorf("UpdatePartial() error = %v", err)
	}

	updated, _ := store.FindByID(ctx, userID)
	if updated.Email != "newemail@example.com" {
		t.Errorf("email not updated: got %s, want newemail@example.com", updated.Email)
	}
	if updated.Name != "Test User" {
		t.Errorf("name should not change: got %s, want Test User", updated.Name)
	}
}

func TestMockUserStoreUpdatePartialName(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)
	userID := user.ID

	req := &UpdateUserRequest{
		Name: NewJsonNull("Updated Name"),
	}

	err := store.UpdatePartial(ctx, userID, req)
	if err != nil {
		t.Errorf("UpdatePartial() error = %v", err)
	}

	updated, _ := store.FindByID(ctx, userID)
	if updated.Name != "Updated Name" {
		t.Errorf("name not updated: got %s, want Updated Name", updated.Name)
	}
	if updated.Email != "test@example.com" {
		t.Errorf("email should not change: got %s, want test@example.com", updated.Email)
	}
}

func TestMockUserStoreUpdatePartialMultipleFields(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)
	userID := user.ID

	req := &UpdateUserRequest{
		Email: NewJsonNull("newemail@example.com"),
		Name:  NewJsonNull("New Name"),
	}

	err := store.UpdatePartial(ctx, userID, req)
	if err != nil {
		t.Errorf("UpdatePartial() error = %v", err)
	}

	updated, _ := store.FindByID(ctx, userID)
	if updated.Email != "newemail@example.com" {
		t.Errorf("email not updated: got %s, want newemail@example.com", updated.Email)
	}
	if updated.Name != "New Name" {
		t.Errorf("name not updated: got %s, want New Name", updated.Name)
	}
}

func TestMockUserStoreUpdatePartialEmpty(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)
	userID := user.ID

	req := &UpdateUserRequest{}

	err := store.UpdatePartial(ctx, userID, req)
	if err != nil {
		t.Errorf("UpdatePartial() error = %v", err)
	}

	updated, _ := store.FindByID(ctx, userID)
	if updated.Email != "test@example.com" {
		t.Errorf("email should not change")
	}
	if updated.Name != "Test User" {
		t.Errorf("name should not change")
	}
}

func TestMockUserStoreUpdatePartialNotPresent(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)
	userID := user.ID

	// Create a request with fields that are present but not valid (null)
	req := &UpdateUserRequest{
		Email: NewJsonNullNull[string](),
	}

	err := store.UpdatePartial(ctx, userID, req)
	if err != nil {
		t.Errorf("UpdatePartial() error = %v", err)
	}

	updated, _ := store.FindByID(ctx, userID)
	if updated.Email != "test@example.com" {
		t.Errorf("email should not change when null: got %s", updated.Email)
	}
}

func TestMockUserStoreUpdatePartialPassword(t *testing.T) {
	store := NewMockUserStore()
	ctx := context.Background()

	user := &User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed_password",
	}

	store.Create(ctx, user)
	userID := user.ID
	oldPassword := user.Password

	req := &UpdateUserRequest{
		Password: NewJsonNull("newpassword123"),
	}

	err := store.UpdatePartial(ctx, userID, req)
	if err != nil {
		t.Errorf("UpdatePartial() error = %v", err)
	}

	updated, _ := store.FindByID(ctx, userID)
	if updated.Password == oldPassword {
		t.Errorf("password should be updated")
	}
	if updated.Password == "newpassword123" {
		t.Errorf("password should be hashed, not stored as plaintext")
	}
}
