package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"url_shortener/internal/config"
	"url_shortener/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateURL(ctx context.Context, req database.CreateURLRequest) (*database.URL, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.URL), args.Error(1)
}

func (m *MockDatabase) GetURLByID(ctx context.Context, id uuid.UUID) (*database.URL, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.URL), args.Error(1)
}

func (m *MockDatabase) GetURLByShortPath(ctx context.Context, shortPath string) (*database.URL, error) {
	args := m.Called(ctx, shortPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.URL), args.Error(1)
}

func (m *MockDatabase) ListURLs(ctx context.Context, page, limit int) (*database.ListURLsResponse, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.ListURLsResponse), args.Error(1)
}

func (m *MockDatabase) UpdateURL(ctx context.Context, id uuid.UUID, req database.UpdateURLRequest) (*database.URL, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.URL), args.Error(1)
}

func (m *MockDatabase) DeleteURL(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabase) PingContext(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockCache struct {
	mock.Mock
}

func (m *MockCache) GetURL(ctx context.Context, shortPath string) (*database.URL, error) {
	args := m.Called(ctx, shortPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.URL), args.Error(1)
}

func (m *MockCache) SetURL(ctx context.Context, shortPath string, url *database.URL) error {
	args := m.Called(ctx, shortPath, url)
	return args.Error(0)
}

func (m *MockCache) DeleteURL(ctx context.Context, shortPath string) error {
	args := m.Called(ctx, shortPath)
	return args.Error(0)
}

func (m *MockCache) GetURLByID(ctx context.Context, id string) (*database.URL, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.URL), args.Error(1)
}

func (m *MockCache) SetURLByID(ctx context.Context, id string, url *database.URL) error {
	args := m.Called(ctx, id, url)
	return args.Error(0)
}

func (m *MockCache) DeleteURLByID(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func setupTestHandler() (*Handler, *MockDatabase, *MockCache) {
	mockDB := new(MockDatabase)
	mockCache := new(MockCache)
	cfg := &config.Config{
		TwitterDomain: "test.com",
	}

	// Create handler without template (skip file parsing for tests)
	handler := &Handler{
		db:     mockDB,
		cache:  mockCache,
		config: cfg,
		tmpl:   nil, // Skip template for unit tests
	}

	return handler, mockDB, mockCache
}

func TestHealthCheck(t *testing.T) {
	handler, mockDB, mockCache := setupTestHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", handler.HealthCheck)

	t.Run("HealthyStatus", func(t *testing.T) {
		mockDB.On("PingContext", mock.Anything).Return(nil)
		mockCache.On("Ping", mock.Anything).Return(nil)

		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])

		mockDB.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("UnhealthyDatabase", func(t *testing.T) {
		// Create fresh mocks to avoid conflicts
		mockDB := new(MockDatabase)
		mockCache := new(MockCache)
		handler := &Handler{
			db:     mockDB,
			cache:  mockCache,
			config: &config.Config{},
		}
		router := gin.New()
		router.GET("/health", handler.HealthCheck)

		mockDB.On("PingContext", mock.Anything).Return(assert.AnError)
		// Cache won't be called since DB fails first

		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		mockDB.AssertExpectations(t)
	})
}

func TestCreateURL(t *testing.T) {
	handler, mockDB, mockCache := setupTestHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/urls", handler.CreateURL)

	t.Run("CreateURLSuccess", func(t *testing.T) {
		testID := uuid.New()
		testTime := time.Now()
		title := "Test Title"

		expectedURL := &database.URL{
			ID:          testID,
			ShortPath:   "abc123",
			Destination: "https://example.com",
			Title:       &title,
			CreatedAt:   testTime,
			UpdatedAt:   testTime,
		}

		mockDB.On("CreateURL", mock.Anything, mock.MatchedBy(func(req database.CreateURLRequest) bool {
			return req.Destination == "https://example.com" && *req.Title == "Test Title"
		})).Return(expectedURL, nil)

		mockCache.On("SetURL", mock.Anything, "abc123", expectedURL).Return(nil)
		mockCache.On("SetURLByID", mock.Anything, testID.String(), expectedURL).Return(nil)

		requestBody := database.CreateURLRequest{
			Destination: "https://example.com",
			Title:       &title,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/urls", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response database.URL
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedURL.ID, response.ID)
		assert.Equal(t, expectedURL.ShortPath, response.ShortPath)
		assert.Equal(t, expectedURL.Destination, response.Destination)

		mockDB.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("CreateURLInvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/urls", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("CreateURLMissingDestination", func(t *testing.T) {
		requestBody := database.CreateURLRequest{
			Title: stringPtr("Test Title"),
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/urls", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetURL(t *testing.T) {
	handler, mockDB, mockCache := setupTestHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/urls/:id", handler.GetURL)

	t.Run("GetURLSuccess", func(t *testing.T) {
		testID := uuid.New()
		expectedURL := &database.URL{
			ID:          testID,
			ShortPath:   "abc123",
			Destination: "https://example.com",
		}

		mockCache.On("GetURLByID", mock.Anything, testID.String()).Return(nil, assert.AnError) // Cache miss
		mockDB.On("GetURLByID", mock.Anything, testID).Return(expectedURL, nil)
		mockCache.On("SetURLByID", mock.Anything, testID.String(), expectedURL).Return(nil)
		mockCache.On("SetURL", mock.Anything, "abc123", expectedURL).Return(nil)

		req, _ := http.NewRequest("GET", "/urls/"+testID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response database.URL
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedURL.ID, response.ID)

		mockDB.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("GetURLNotFound", func(t *testing.T) {
		testID := uuid.New()

		mockCache.On("GetURLByID", mock.Anything, testID.String()).Return(nil, assert.AnError) // Cache miss
		mockDB.On("GetURLByID", mock.Anything, testID).Return(nil, nil) // Not found

		req, _ := http.NewRequest("GET", "/urls/"+testID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockDB.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("GetURLInvalidID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/urls/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestListURLs(t *testing.T) {
	handler, mockDB, _ := setupTestHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/urls", handler.ListURLs)

	t.Run("ListURLsSuccess", func(t *testing.T) {
		expectedResponse := &database.ListURLsResponse{
			URLs: []database.URL{
				{ID: uuid.New(), ShortPath: "abc123", Destination: "https://example.com"},
				{ID: uuid.New(), ShortPath: "def456", Destination: "https://test.com"},
			},
			Total: 2,
			Page:  1,
			Limit: 10,
		}

		mockDB.On("ListURLs", mock.Anything, 1, 10).Return(expectedResponse, nil)

		req, _ := http.NewRequest("GET", "/urls", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response database.ListURLsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedResponse.Total, response.Total)
		assert.Len(t, response.URLs, 2)

		mockDB.AssertExpectations(t)
	})

	t.Run("ListURLsWithPagination", func(t *testing.T) {
		expectedResponse := &database.ListURLsResponse{
			URLs:  []database.URL{},
			Total: 25,
			Page:  2,
			Limit: 5,
		}

		mockDB.On("ListURLs", mock.Anything, 2, 5).Return(expectedResponse, nil)

		req, _ := http.NewRequest("GET", "/urls?page=2&limit=5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response database.ListURLsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 2, response.Page)
		assert.Equal(t, 5, response.Limit)

		mockDB.AssertExpectations(t)
	})
}

func TestDeleteURL(t *testing.T) {
	handler, mockDB, mockCache := setupTestHandler()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/urls/:id", handler.DeleteURL)

	t.Run("DeleteURLSuccess", func(t *testing.T) {
		testID := uuid.New()
		testURL := &database.URL{
			ID:        testID,
			ShortPath: "abc123",
		}

		mockDB.On("GetURLByID", mock.Anything, testID).Return(testURL, nil)
		mockDB.On("DeleteURL", mock.Anything, testID).Return(nil)
		mockCache.On("DeleteURL", mock.Anything, "abc123").Return(nil)
		mockCache.On("DeleteURLByID", mock.Anything, testID.String()).Return(nil)

		req, _ := http.NewRequest("DELETE", "/urls/"+testID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		mockDB.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("DeleteURLNotFound", func(t *testing.T) {
		testID := uuid.New()

		mockDB.On("GetURLByID", mock.Anything, testID).Return(nil, nil)

		req, _ := http.NewRequest("DELETE", "/urls/"+testID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		mockDB.AssertExpectations(t)
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}