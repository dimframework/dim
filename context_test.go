package dim

import (
	"net/http"
	"testing"
)

func TestSetAndGetUser(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	user := &TokenUser{
		ID:    "1",
		Email: "test@example.com",
	}

	req = SetUser(req, user)
	retrieved, ok := GetUser(req)

	if !ok {
		t.Errorf("GetUser returned false, expected true")
	}

	if retrieved.GetID() != user.ID || retrieved.GetEmail() != user.Email {
		t.Errorf("retrieved user mismatch: got %+v, want %+v", retrieved, user)
	}
}

func TestGetUserNotSet(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	user, ok := GetUser(req)

	if ok {
		t.Errorf("GetUser should return false when user not set")
	}

	if user != nil {
		t.Errorf("user should be nil when not set")
	}
}

func TestSetAndGetRequestID(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	requestID := "12345"

	req = SetRequestID(req, requestID)
	retrieved := GetRequestID(req)

	if retrieved != requestID {
		t.Errorf("GetRequestID = %s, want %s", retrieved, requestID)
	}
}

func TestGetRequestIDNotSet(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	requestID := GetRequestID(req)

	if requestID != "" {
		t.Errorf("GetRequestID should return empty string when not set")
	}
}

func TestGetQueryParam(t *testing.T) {
	req, _ := http.NewRequest("GET", "/?page=1&limit=10", nil)

	if GetQueryParam(req, "page") != "1" {
		t.Errorf("GetQueryParam('page') failed")
	}

	if GetQueryParam(req, "limit") != "10" {
		t.Errorf("GetQueryParam('limit') failed")
	}
}

func TestGetQueryParamNotSet(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)

	if GetQueryParam(req, "page") != "" {
		t.Errorf("GetQueryParam should return empty string when not set")
	}
}

func TestGetQueryParams(t *testing.T) {
	req, _ := http.NewRequest("GET", "/?page=1&limit=10&sort=name", nil)

	params := GetQueryParams(req, "page", "limit", "sort")

	if params["page"] != "1" || params["limit"] != "10" || params["sort"] != "name" {
		t.Errorf("GetQueryParams mismatch: got %v", params)
	}
}

func TestGetHeaderValue(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Custom-Header", "custom-value")

	if GetHeaderValue(req, "X-Custom-Header") != "custom-value" {
		t.Errorf("GetHeaderValue failed")
	}
}

func TestGetAuthToken(t *testing.T) {
	tests := []struct {
		name      string
		authHead  string
		wantToken string
		wantOk    bool
	}{
		{
			name:      "valid bearer token",
			authHead:  "Bearer eyJhbGciOiJIUzI1NiIs",
			wantToken: "eyJhbGciOiJIUzI1NiIs",
			wantOk:    true,
		},
		{
			name:      "missing Bearer",
			authHead:  "eyJhbGciOiJIUzI1NiIs",
			wantToken: "",
			wantOk:    false,
		},
		{
			name:      "empty header",
			authHead:  "",
			wantToken: "",
			wantOk:    false,
		},
		{
			name:      "invalid format",
			authHead:  "Bearer",
			wantToken: "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.authHead != "" {
				req.Header.Set("Authorization", tt.authHead)
			}

			token, ok := GetAuthToken(req)
			if ok != tt.wantOk {
				t.Errorf("GetAuthToken ok = %v, want %v", ok, tt.wantOk)
			}

			if token != tt.wantToken {
				t.Errorf("GetAuthToken token = %s, want %s", token, tt.wantToken)
			}
		})
	}
}
