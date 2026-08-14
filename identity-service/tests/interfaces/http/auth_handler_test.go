package http_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/user"
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

func setupAuthHandler(t *testing.T) (
	*httpui.AuthHandler,
	auth.JWTManager,
	auth.RefreshTokenRepository,
	auth.RefreshTokenManager,
) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableEmailVerification, testutil.TableUser)

	_ = fixtures.LoadUserFixtures(t, db.DB)

	rdb := testutil.Redis(t)
	rdb.Flush(t)

	userRepo := persistence.NewUserRepository(db.DB)
	hasher := security.NewPasswordHasher()
	jwt := security.NewJWTManager("TestSecretKey")
	refreshTokenRepo := persistence.NewRefreshTokenRepository(rdb.Client)
	refreshTokenManager := security.NewRefreshTokenManager()

	login := authapp.NewLogin(userRepo, hasher, jwt, refreshTokenRepo, refreshTokenManager)
	refreshToken := authapp.NewRefreshToken(jwt, refreshTokenRepo, refreshTokenManager)
	logout := authapp.NewLogout(refreshTokenRepo, refreshTokenManager)

	handler := httpui.NewAuthHandler(login, nil, refreshToken, logout)

	return handler, jwt, refreshTokenRepo, refreshTokenManager
}

func TestAuthHandler_Login_Success(t *testing.T) {
	handler, jwt, _, _ := setupAuthHandler(t)

	tc := struct {
		name         string
		requestBody  map[string]string
		expectedCode int
		expectedBody string
	}{
		name: "Successful Login",
		requestBody: map[string]string{
			"email":    "existing@example.com", // from fixture
			"password": "plainPassword",
		},
		expectedCode: http.StatusOK,
	}

	t.Run(tc.name, func(t *testing.T) {
		router := gin.Default()
		router.POST("/auth/login", handler.Login)

		body, _ := json.Marshal(tc.requestBody)
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		type response struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		}

		assert.Equal(t, tc.expectedCode, w.Code)

		var res response
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.NoError(t, err)

		_, err = jwt.Parse(res.AccessToken)
		assert.NoError(t, err)

		_, err = hex.DecodeString(res.RefreshToken)
		assert.NoError(t, err)
	})
}

func TestAuthHandler_Login_Failed(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)

	tests := []struct {
		name         string
		requestBody  map[string]string
		expectedCode int
		expectedBody string
	}{
		{
			name: "Successful Login",
			requestBody: map[string]string{
				"email":    "existing@example.com", // from fixture
				"password": "wrongpassword",
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: `{"error":"unauthorized access"}`,
		},
		{
			name: "Invalid Credentials",
			requestBody: map[string]string{
				"email":    "no.user.found@example.com",
				"password": "random",
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: `{"error":"unauthorized access"}`,
		},
		{
			name: "Invalid Request Body",
			requestBody: map[string]string{
				"email":    "invalid-email.com",
				"password": "password123",
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"errors":[{"field":"Email","message":"Please provide a valid email address."}]}`,
		},
		{
			name: "Invalid Request Body",
			requestBody: map[string]string{
				"email":    "test@example.com",
				"password": "",
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"errors":[{"field":"Password","message":"This field is required."}]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.Default()
			router.POST("/auth/login", handler.Login)

			body, _ := json.Marshal(tc.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			assert.Equal(t, tc.expectedCode, recorder.Code)
			assert.JSONEq(t, tc.expectedBody, recorder.Body.String())
		})
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	handler, _, repo, manager := setupAuthHandler(t)

	router := gin.Default()
	router.POST("/auth/refresh", handler.Refresh)

	rawToken, err := manager.Generate()
	require.NoError(t, err)

	hashed := manager.Hash(rawToken)
	require.NoError(t, err)

	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	err = repo.Save(context.Background(), hashed, claims, ttl)
	require.NoError(t, err)

	body, _ := json.Marshal(authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var res authapp.TokenResponse
	err = json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)

	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
	assert.NotEqual(t, rawToken, res.RefreshToken)
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)

	router := gin.Default()
	router.POST("/auth/refresh", handler.Refresh)

	body, _ := json.Marshal(authapp.RefreshRequest{
		RefreshToken: "invalid-token",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)

	var res map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)

	assert.Equal(t, "refresh token not found or expired", res["error"])
}

func TestAuthHandler_Refresh_InvalidRequest(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)

	router := gin.Default()
	router.POST("/auth/refresh", handler.Refresh)

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/refresh",
		bytes.NewReader([]byte(`{}`)), // missing refreshToken
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var res map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)

	assert.Equal(t, "invalid request", res["error"])
}

func TestAuthHandler_Refresh_Rotation(t *testing.T) {
	handler, _, repo, manager := setupAuthHandler(t)

	router := gin.Default()
	router.POST("/auth/refresh", handler.Refresh)

	rawToken, _ := manager.Generate()
	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	_ = repo.Save(context.Background(), hashed, claims, ttl)

	body, _ := json.Marshal(authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	req1 := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	require.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	ctx := context.Background()
	db := testutil.DB(t)

	handler, _, repo, manager := setupAuthHandler(t)

	var u user.User
	err := db.DB.Where("email = ?", "existing@example.com").First(&u).Error // from fixture
	require.NoError(t, err)

	router := gin.Default()
	router.Use(MockAuthMiddleware(u.ID.String(), u.Role))
	router.POST("/auth/logout", handler.Logout)

	rawToken, _ := manager.Generate()
	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: u.ID.String(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	_ = repo.Save(ctx, hashed, claims, ttl)

	body, _ := json.Marshal(authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// verify deleted
	_, err = repo.Find(ctx, hashed)
	require.Error(t, err)
}

func TestAuthHandler_Logout_Idempotent(t *testing.T) {
	ctx := context.Background()
	db := testutil.DB(t)

	handler, _, repo, manager := setupAuthHandler(t)

	var u user.User
	err := db.DB.Where("email = ?", "existing@example.com").First(&u).Error // from fixture
	require.NoError(t, err)

	router := gin.Default()
	router.Use(MockAuthMiddleware(u.ID.String(), u.Role))
	router.POST("/auth/logout", handler.Logout)

	rawToken, _ := manager.Generate()
	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: u.ID.String(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	_ = repo.Save(ctx, hashed, claims, ttl)

	body, _ := json.Marshal(authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	// First call
	req1 := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer mock-valid-token")

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	require.Equal(t, http.StatusOK, w1.Code)

	// Second call
	req2 := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer mock-valid-token")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
}

func TestAuthHandler_Logout_InvalidRequest(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)
	db := testutil.DB(t)

	var u user.User
	err := db.DB.Where("email = ?", "existing@example.com").First(&u).Error // from fixture
	require.NoError(t, err)

	router := gin.Default()
	router.Use(MockAuthMiddleware(u.ID.String(), u.Role))
	router.POST("/auth/logout", handler.Logout)

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/logout",
		bytes.NewReader([]byte(`{}`)), // missing refreshToken
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-valid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
