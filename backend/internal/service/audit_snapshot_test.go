package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"translator-checkin/internal/model"
)

func TestSnapshotUser_RedactsSensitiveFields(t *testing.T) {
	u := &model.User{
		ID:           7,
		Email:        "a@b.com",
		Name:         "Alice",
		Phone:        "0900",
		Role:         "admin",
		Status:       "active",
		PasswordHash: "super-secret-hash",
	}
	snap := snapshotUser(u)

	assert.Equal(t, uint(7), snap["id"])
	assert.Equal(t, "a@b.com", snap["email"])
	assert.Equal(t, "admin", snap["role"])
	// Sensitive fields must never leak into the audit detail.
	assert.NotContains(t, snap, "password_hash")
	assert.NotContains(t, snap, "PasswordHash")

	// Marshalling the whole snapshot must not contain the hash anywhere.
	b, err := json.Marshal(snap)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "super-secret-hash")
}

func TestSnapshotUser_Nil(t *testing.T) {
	assert.Nil(t, snapshotUser(nil))
}

func TestAuditDetailJSON_DeleteHasBeforeOnly(t *testing.T) {
	detail := auditDetailJSON(map[string]any{"id": 1, "name": "X"}, nil)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(detail), &parsed))
	assert.Contains(t, parsed, "before")
	// "after" is omitempty, so a delete (after=nil) must not include the key.
	assert.NotContains(t, parsed, "after")
}

func TestAuditDetailJSON_UpdateHasBeforeAndAfter(t *testing.T) {
	detail := auditDetailJSON(
		map[string]any{"name": "Old"},
		map[string]any{"name": "New"},
	)

	var parsed struct {
		Before map[string]any `json:"before"`
		After  map[string]any `json:"after"`
	}
	require.NoError(t, json.Unmarshal([]byte(detail), &parsed))
	assert.Equal(t, "Old", parsed.Before["name"])
	assert.Equal(t, "New", parsed.After["name"])
}
