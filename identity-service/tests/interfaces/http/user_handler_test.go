package http_test

import (
	"bytes"
	"encoding/json"
	userapp "identity-service/internal/application/user"
	"identity-service/internal/domain/user"
	"identity-service/internal/infrastructure/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/internal/infrastructure/security"
	httpui "identity-service/internal/interfaces/http"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserHandler(t *testing.T) *httpui.UserHandler {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableEmailVerification, testutil.TableUser)

	_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)
	_ = fixtures.LoadUserFixtures(t, db.DB)

	emailVerificationRepo := persistence.NewEmailVerificationRepository(db.DB)
	codeVerifier := auth.NewEmailVerifier(emailVerificationRepo)
	hasher := security.NewPasswordHasher()
	userRepo := persistence.NewUserRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	register := userapp.NewRegisterCustomer(db.DB, codeVerifier, userRepo, hasher, outboxRepo)
	registerOwner := userapp.NewRegisterOwner(db.DB, codeVerifier, hasher, userRepo, outboxRepo)
	findByID := userapp.NewFindByID(userRepo)

	return httpui.NewUserHandler(register, registerOwner, findByID)
}

func TestUserHandler_RegisterOwner(t *testing.T) {
	handler := setupUserHandler(t)

	tests := []struct {
		name           string
		body           map[string]string
		rawBody        string
		expectedStatus int
		expectError    bool
	}{
		{
			name: "success",
			body: map[string]string{
				"firstName":    "Alice",
				"lastName":     "Doe",
				"email":        "alice@example.com",
				"password":     "pass123",
				"code":         "347578", // from fixture
				"businessName": "Domino's Pizza",
				"vatNumber":    "DE123456789",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid json",
			rawBody:        "invalid-json",
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
		{
			name: "invalid code",
			body: map[string]string{
				"firstName":    "Alice",
				"lastName":     "Doe",
				"email":        "alice@example.com",
				"password":     "pass123",
				"code":         "000000", // invalid
				"businessName": "Domino's Pizza",
				"vatNumber":    "DE987654321",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.DB(t)
			db.TruncateTables(
				t,
				testutil.TableEmailVerification,
				testutil.TableUser,
				testutil.TableOutboxEvent,
			)

			_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)
			_ = fixtures.LoadUserFixtures(t, db.DB)

			router := gin.Default()
			router.POST("/users/owners", handler.RegisterOwner)

			var reqBody []byte
			if tt.rawBody != "" {
				reqBody = []byte(tt.rawBody)
			} else {
				reqBody, _ = json.Marshal(tt.body)
			}

			req, _ := http.NewRequest("POST", "/users/owners", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestUserHandler_RegisterCustomer(t *testing.T) {
	handler := setupUserHandler(t)

	tests := []struct {
		name           string
		body           map[string]string
		rawBody        string
		expectedStatus int
		expectError    bool
	}{
		{
			name: "success",
			body: map[string]string{
				"firstName": "Sophie",
				"lastName":  "Mueller",
				"email":     "sophie.mueller@example.com",
				"password":  "pass123",
				"code":      "365189", // from fixture
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "validation error",
			body:           map[string]string{},
			expectedStatus: http.StatusUnprocessableEntity,
			expectError:    true,
		},
		{
			name: "duplicate email",
			body: map[string]string{
				"firstName": "Existing",
				"lastName":  "User",
				"email":     "existing@example.com", // from fixture
				"password":  "pass123",
				"code":      "347578",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.DB(t)
			db.TruncateTables(
				t,
				testutil.TableEmailVerification,
				testutil.TableUser,
				testutil.TableOutboxEvent,
			)

			_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)
			_ = fixtures.LoadUserFixtures(t, db.DB)

			router := gin.Default()
			router.POST("/users/customers", handler.RegisterCustomer)

			var reqBody []byte
			if tt.rawBody != "" {
				reqBody = []byte(tt.rawBody)
			} else {
				reqBody, _ = json.Marshal(tt.body)
			}

			req, _ := http.NewRequest("POST", "/users/customers", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestUserHandler_FindByID_Success(t *testing.T) {
	handler := setupUserHandler(t)
	db := testutil.DB(t)

	var u user.User
	err := db.DB.Where("email = ?", "existing@example.com").First(&u).Error // from fixture
	require.NoError(t, err)

	router := gin.Default()
	router.Use(MockAuthMiddleware(u.ID.String(), u.Role))
	router.GET("/users/:id", handler.FindByID)

	req, _ := http.NewRequest(http.MethodGet, "/users/"+u.ID.String(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var res userapp.Response
	err = json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)

	assert.Equal(t, u.ID, res.ID)
	assert.Equal(t, u.FirstName, res.Name.First)
	assert.Equal(t, u.LastName, res.Name.Last)
	assert.Equal(t, u.Email, res.Email)
	assert.Equal(t, u.Role, res.Role)
	assert.Equal(t, u.Status, res.Status)
}

func TestUserHandler_FindByID_NotFound(t *testing.T) {
	handler := setupUserHandler(t)
	db := testutil.DB(t)

	var u user.User
	err := db.DB.Where("email = ?", "existing@example.com").First(&u).Error // from fixture
	require.NoError(t, err)

	router := gin.Default()
	router.Use(MockAuthMiddleware(u.ID.String(), u.Role))
	router.GET("/users/:id", handler.FindByID)

	newID := testutil.MustNewID()

	req := httptest.NewRequest(http.MethodGet, "/users/"+newID.String(), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "resource not found", body["error"])
}

func TestUserHandler_FindByID_Failure_Unauthorized(t *testing.T) {
	handler := setupUserHandler(t)
	db := testutil.DB(t)

	var u user.User
	err := db.DB.Where("email = ?", "existing@example.com").First(&u).Error // from fixture
	require.NoError(t, err)

	router := gin.Default()
	router.Use(MockAuthMiddleware(u.ID.String(), u.Role))
	router.GET("/users/:id", handler.FindByID)

	req, _ := http.NewRequest(http.MethodGet, "/users/"+u.ID.String(), nil)
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}
