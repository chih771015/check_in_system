package service

import (
	"context"
	"errors"
	"testing"

	"translator-checkin/internal/dto"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTranslatorService(t *testing.T) (*TranslatorService, *repository.UserRepository) {
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	return NewTranslatorService(repo), repo
}

func TestTranslatorService_List_FiltersByStatus(t *testing.T) {
	svc, repo := newTranslatorService(t)
	require.NoError(t, repo.Create(&model.User{Email: "a@x.com", PasswordHash: "h", Name: "A", Role: "translator", Status: "active"}))
	require.NoError(t, repo.Create(&model.User{Email: "b@x.com", PasswordHash: "h", Name: "B", Role: "translator", Status: "disabled"}))
	require.NoError(t, repo.Create(&model.User{Email: "admin@x.com", PasswordHash: "h", Name: "Admin", Role: "admin", Status: "active"}))

	all, total, err := svc.List(context.Background(), "", 0, 0)
	require.NoError(t, err)
	assert.Len(t, all, 2, "should not include admin")
	assert.Equal(t, int64(2), total)

	active, activeTotal, err := svc.List(context.Background(), "active", 0, 0)
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, int64(1), activeTotal)
	assert.Equal(t, "A", active[0].Name)

	// With a page size the rows are capped but total still reflects the full count.
	page1, pTotal, err := svc.List(context.Background(), "", 1, 1)
	require.NoError(t, err)
	assert.Len(t, page1, 1, "page size caps rows")
	assert.Equal(t, int64(2), pTotal)
}

func TestTranslatorService_Create_DuplicateEmail(t *testing.T) {
	svc, repo := newTranslatorService(t)
	require.NoError(t, repo.Create(&model.User{Email: "dup@x.com", PasswordHash: "h", Name: "X", Role: "translator", Status: "active"}))

	id, err := svc.Create(context.Background(), dto.CreateTranslatorRequest{
		Email: "dup@x.com", Password: "pass1234", Name: "Y", Phone: "1",
	})
	assert.True(t, errors.Is(err, ErrEmailTaken))
	assert.Zero(t, id, "failed create should return zero id")
}

func TestTranslatorService_Create_Success_HashesAndForcesPasswordChange(t *testing.T) {
	svc, repo := newTranslatorService(t)
	id, err := svc.Create(context.Background(), dto.CreateTranslatorRequest{
		Email: "new@x.com", Password: "pass1234", Name: "New", Phone: "1",
	})
	require.NoError(t, err)
	assert.NotZero(t, id, "successful create should return the new user id")

	u, err := repo.FindByEmail("new@x.com")
	require.NoError(t, err)
	assert.Equal(t, id, u.ID, "returned id should match the persisted user")
	assert.Equal(t, "translator", u.Role)
	assert.True(t, u.MustChangePW, "new translator must change password on first login")
	// bcrypt 雜湊應可驗證
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("pass1234")))
}

func TestTranslatorService_Update_NotFound(t *testing.T) {
	svc, _ := newTranslatorService(t)
	_, err := svc.Update(context.Background(), 99999, dto.UpdateTranslatorRequest{})
	assert.True(t, errors.Is(err, ErrTranslatorNotFound))
}

func TestTranslatorService_Update_RejectsAdminTarget(t *testing.T) {
	svc, repo := newTranslatorService(t)
	admin := &model.User{Email: "a@x.com", PasswordHash: "h", Name: "A", Role: "admin", Status: "active"}
	require.NoError(t, repo.Create(admin))

	_, err := svc.Update(context.Background(), admin.ID, dto.UpdateTranslatorRequest{})
	assert.True(t, errors.Is(err, ErrNotATranslator))
}

func TestTranslatorService_Update_InvalidStatus(t *testing.T) {
	svc, repo := newTranslatorService(t)
	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active"}
	require.NoError(t, repo.Create(tr))

	bad := "frozen"
	_, err := svc.Update(context.Background(), tr.ID, dto.UpdateTranslatorRequest{Status: &bad})
	assert.True(t, errors.Is(err, ErrInvalidStatus))
}

func TestTranslatorService_Update_PartialFields(t *testing.T) {
	svc, repo := newTranslatorService(t)
	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "OldName", Phone: "111", Role: "translator", Status: "active"}
	require.NoError(t, repo.Create(tr))

	newName := "NewName"
	detail, err := svc.Update(context.Background(), tr.ID, dto.UpdateTranslatorRequest{Name: &newName})
	require.NoError(t, err)
	// Audit detail should capture the before (OldName) and after (NewName).
	assert.Contains(t, detail, "OldName")
	assert.Contains(t, detail, "NewName")

	reloaded, _ := repo.FindByID(tr.ID)
	assert.Equal(t, "NewName", reloaded.Name)
	assert.Equal(t, "111", reloaded.Phone, "untouched phone preserved")
}

func TestTranslatorService_Disable_RejectsAdmin(t *testing.T) {
	svc, repo := newTranslatorService(t)
	admin := &model.User{Email: "a@x.com", PasswordHash: "h", Name: "A", Role: "admin", Status: "active"}
	require.NoError(t, repo.Create(admin))

	err := svc.Disable(context.Background(), admin.ID)
	assert.True(t, errors.Is(err, ErrNotATranslator))
}

func TestTranslatorService_Disable_Success(t *testing.T) {
	svc, repo := newTranslatorService(t)
	tr := &model.User{Email: "t@x.com", PasswordHash: "h", Name: "T", Role: "translator", Status: "active"}
	require.NoError(t, repo.Create(tr))

	require.NoError(t, svc.Disable(context.Background(), tr.ID))
	reloaded, _ := repo.FindByID(tr.ID)
	assert.Equal(t, "disabled", reloaded.Status)
}
