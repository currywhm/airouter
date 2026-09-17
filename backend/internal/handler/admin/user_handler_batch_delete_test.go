package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type batchDeleteUserAdminService struct {
	*stubAdminService
	deletedIDs []int64
	errorsByID map[int64]error
}

func (s *batchDeleteUserAdminService) DeleteUser(_ context.Context, id int64) error {
	s.deletedIDs = append(s.deletedIDs, id)
	return s.errorsByID[id]
}

func TestUserHandlerBatchDeleteReturnsStablePerUserResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := &batchDeleteUserAdminService{
		stubAdminService: newStubAdminService(),
		errorsByID: map[int64]error{
			3: errors.New("delete failed"),
		},
	}
	handler := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/api/v1/admin/users/batch-delete", handler.BatchDelete)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/users/batch-delete",
		bytes.NewBufferString(`{"user_ids":[5,4,3,2,1,2,0,-1]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload struct {
		Data struct {
			Total      int     `json:"total"`
			Deleted    int     `json:"deleted"`
			Failed     int     `json:"failed"`
			DeletedIDs []int64 `json:"deleted_ids"`
			FailedIDs  []int64 `json:"failed_ids"`
			Errors     []struct {
				UserID int64  `json:"user_id"`
				Error  string `json:"error"`
			} `json:"errors"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 5, payload.Data.Total)
	require.Equal(t, 4, payload.Data.Deleted)
	require.Equal(t, 1, payload.Data.Failed)
	require.Equal(t, []int64{1, 2, 4, 5}, payload.Data.DeletedIDs)
	require.Equal(t, []int64{3}, payload.Data.FailedIDs)
	require.Equal(t, int64(3), payload.Data.Errors[0].UserID)
	require.Equal(t, "delete failed", payload.Data.Errors[0].Error)
}
