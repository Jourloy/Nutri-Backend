package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeAdminService struct {
	getAllUsersFn        func(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, int64, error)
	getUserDetailsFn     func(ctx context.Context, userId string) (*UserDetailsResponse, error)
	grantUserSubFn       func(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error)
	deleteUserFn         func(ctx context.Context, userId string) error
	updatePlanPriceFn    func(ctx context.Context, planId int64, amountMinor int64) error
	updateUserPriceFn    func(ctx context.Context, userId string, amountMinor int64) error
	updatePlanFeaturesFn func(ctx context.Context, planId int64, features map[string]interface{}) error
	createNotificationFn func(ctx context.Context, createdBy string, notification *AdminNotificationCreate) (*AdminNotification, error)
	getNotificationsFn   func(ctx context.Context, limit, offset int) ([]AdminNotification, error)
	sendNotificationFn   func(ctx context.Context, notificationId int64) error
	createUserWithSubFn  func(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error
	getDashboardStatsFn  func(ctx context.Context) (*DashboardStats, error)
}

func (f fakeAdminService) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	if f.getDashboardStatsFn != nil {
		return f.getDashboardStatsFn(ctx)
	}
	return &DashboardStats{}, nil
}

func (f fakeAdminService) GetAllUsers(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, int64, error) {
	if f.getAllUsersFn != nil {
		return f.getAllUsersFn(ctx, limit, offset, sortBy, sortOrder)
	}
	return []UserListItem{}, 0, nil
}

func (f fakeAdminService) GetUserDetails(ctx context.Context, userId string) (*UserDetailsResponse, error) {
	if f.getUserDetailsFn != nil {
		return f.getUserDetailsFn(ctx, userId)
	}
	return nil, sql.ErrNoRows
}

func (f fakeAdminService) CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error {
	if f.createUserWithSubFn != nil {
		return f.createUserWithSubFn(ctx, username, passwordHash, email, planId, durationMs)
	}
	return nil
}

func (f fakeAdminService) GrantUserSubscription(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error) {
	if f.grantUserSubFn != nil {
		return f.grantUserSubFn(ctx, userId, planId, durationDays)
	}
	return nil, sql.ErrNoRows
}

func (f fakeAdminService) DeleteUser(ctx context.Context, userId string) error {
	if f.deleteUserFn != nil {
		return f.deleteUserFn(ctx, userId)
	}
	return nil
}

func (f fakeAdminService) UpdatePlanPrice(ctx context.Context, planId int64, amountMinor int64) error {
	if f.updatePlanPriceFn != nil {
		return f.updatePlanPriceFn(ctx, planId, amountMinor)
	}
	return nil
}

func (f fakeAdminService) UpdateUserSubscriptionPrice(ctx context.Context, userId string, amountMinor int64) error {
	if f.updateUserPriceFn != nil {
		return f.updateUserPriceFn(ctx, userId, amountMinor)
	}
	return nil
}

func (f fakeAdminService) UpdatePlanFeatures(ctx context.Context, planId int64, features map[string]interface{}) error {
	if f.updatePlanFeaturesFn != nil {
		return f.updatePlanFeaturesFn(ctx, planId, features)
	}
	return nil
}

func (f fakeAdminService) CreateNotification(ctx context.Context, createdBy string, notification *AdminNotificationCreate) (*AdminNotification, error) {
	if f.createNotificationFn != nil {
		return f.createNotificationFn(ctx, createdBy, notification)
	}
	return &AdminNotification{}, nil
}

func (f fakeAdminService) GetNotifications(ctx context.Context, limit, offset int) ([]AdminNotification, error) {
	if f.getNotificationsFn != nil {
		return f.getNotificationsFn(ctx, limit, offset)
	}
	return []AdminNotification{}, nil
}

func (f fakeAdminService) SendNotification(ctx context.Context, notificationId int64) error {
	if f.sendNotificationFn != nil {
		return f.sendNotificationFn(ctx, notificationId)
	}
	return nil
}

func newAdminControllerForTests(service Service) *Controller {
	return &Controller{service: service}
}

func TestGetUsers_DefaultSort(t *testing.T) {
	var gotSortBy UserSortBy
	var gotSortOrder SortOrder

	controller := newAdminControllerForTests(fakeAdminService{
		getAllUsersFn: func(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, int64, error) {
			gotSortBy = sortBy
			gotSortOrder = sortOrder
			return []UserListItem{}, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users?limit=20&offset=0", nil)
	rr := httptest.NewRecorder()
	controller.GetUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if gotSortBy != UserSortByCreatedAt || gotSortOrder != SortOrderDesc {
		t.Fatalf("expected created_at/desc, got %s/%s", gotSortBy, gotSortOrder)
	}
}

func TestGetUsers_ExplicitSort(t *testing.T) {
	var gotSortBy UserSortBy
	var gotSortOrder SortOrder

	controller := newAdminControllerForTests(fakeAdminService{
		getAllUsersFn: func(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, int64, error) {
			gotSortBy = sortBy
			gotSortOrder = sortOrder
			return []UserListItem{}, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users?sort_by=username&sort_order=asc", nil)
	rr := httptest.NewRecorder()
	controller.GetUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if gotSortBy != UserSortByUsername || gotSortOrder != SortOrderAsc {
		t.Fatalf("expected username/asc, got %s/%s", gotSortBy, gotSortOrder)
	}
}

func TestGetUsers_InvalidSortFallback(t *testing.T) {
	var gotSortBy UserSortBy
	var gotSortOrder SortOrder

	controller := newAdminControllerForTests(fakeAdminService{
		getAllUsersFn: func(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, int64, error) {
			gotSortBy = sortBy
			gotSortOrder = sortOrder
			return []UserListItem{}, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users?sort_by=boom&sort_order=boom", nil)
	rr := httptest.NewRecorder()
	controller.GetUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if gotSortBy != UserSortByCreatedAt || gotSortOrder != SortOrderDesc {
		t.Fatalf("expected fallback created_at/desc, got %s/%s", gotSortBy, gotSortOrder)
	}
}

func TestGetUserDetails_Success(t *testing.T) {
	controller := newAdminControllerForTests(fakeAdminService{
		getUserDetailsFn: func(ctx context.Context, userId string) (*UserDetailsResponse, error) {
			return &UserDetailsResponse{
				User: &AdminUserProfile{Id: userId, Username: "john"},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users/u-1", nil)
	req = withURLParam(req, "userId", "u-1")
	rr := httptest.NewRecorder()
	controller.GetUserDetails(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var out UserDetailsResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if out.User == nil || out.User.Id != "u-1" {
		t.Fatalf("unexpected user payload: %+v", out.User)
	}
}

func TestGetUserDetails_NotFound(t *testing.T) {
	controller := newAdminControllerForTests(fakeAdminService{
		getUserDetailsFn: func(ctx context.Context, userId string) (*UserDetailsResponse, error) {
			return nil, sql.ErrNoRows
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/users/u-404", nil)
	req = withURLParam(req, "userId", "u-404")
	rr := httptest.NewRecorder()
	controller.GetUserDetails(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestGrantUserSubscription_Success(t *testing.T) {
	var gotUserID string
	var gotPlanID int64
	var gotDuration int64

	controller := newAdminControllerForTests(fakeAdminService{
		grantUserSubFn: func(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error) {
			gotUserID = userId
			gotPlanID = planId
			gotDuration = durationDays
			return &AdminUserSubscription{
				Id:     42,
				UserId: userId,
				PlanId: planId,
				Status: "active",
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/users/u-1/subscription/grant", strings.NewReader(`{"plan_id":2,"duration_days":30}`))
	req = withURLParam(req, "userId", "u-1")
	rr := httptest.NewRecorder()
	controller.GrantUserSubscription(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if gotUserID != "u-1" || gotPlanID != 2 || gotDuration != 30 {
		t.Fatalf("unexpected service args: user=%s plan=%d duration=%d", gotUserID, gotPlanID, gotDuration)
	}
}

func TestGrantUserSubscription_InvalidPayload(t *testing.T) {
	controller := newAdminControllerForTests(fakeAdminService{})

	req := httptest.NewRequest(http.MethodPost, "/admin/users/u-1/subscription/grant", strings.NewReader(`{"plan_id":0,"duration_days":0}`))
	req = withURLParam(req, "userId", "u-1")
	rr := httptest.NewRecorder()
	controller.GrantUserSubscription(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGrantUserSubscription_NotFound(t *testing.T) {
	controller := newAdminControllerForTests(fakeAdminService{
		grantUserSubFn: func(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error) {
			return nil, sql.ErrNoRows
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/users/u-404/subscription/grant", strings.NewReader(`{"plan_id":2,"duration_days":30}`))
	req = withURLParam(req, "userId", "u-404")
	rr := httptest.NewRecorder()
	controller.GrantUserSubscription(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	var deletedUserID string
	controller := newAdminControllerForTests(fakeAdminService{
		deleteUserFn: func(ctx context.Context, userId string) error {
			deletedUserID = userId
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/u-1", nil)
	req = withURLParam(req, "userId", "u-1")
	rr := httptest.NewRecorder()
	controller.DeleteUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if deletedUserID != "u-1" {
		t.Fatalf("expected user u-1 to be deleted, got %s", deletedUserID)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	controller := newAdminControllerForTests(fakeAdminService{
		deleteUserFn: func(ctx context.Context, userId string) error {
			return sql.ErrNoRows
		},
	})

	req := httptest.NewRequest(http.MethodDelete, "/admin/users/u-404", nil)
	req = withURLParam(req, "userId", "u-404")
	rr := httptest.NewRecorder()
	controller.DeleteUser(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
