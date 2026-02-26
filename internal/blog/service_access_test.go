package blog

import (
	"slices"
	"testing"
)

func TestCanAccessArticle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status string
		viewer ViewerAccess
		want   bool
	}{
		{
			name:   "anonymous can read public",
			status: "public",
			viewer: ViewerAccess{},
			want:   true,
		},
		{
			name:   "anonymous cannot read authorized",
			status: "authorized",
			viewer: ViewerAccess{},
			want:   false,
		},
		{
			name:   "authenticated start can read authorized",
			status: "authorized",
			viewer: ViewerAccess{IsAuthenticated: true, PlanCode: "START"},
			want:   true,
		},
		{
			name:   "authenticated start cannot read paid",
			status: "paid",
			viewer: ViewerAccess{IsAuthenticated: true, PlanCode: "START"},
			want:   false,
		},
		{
			name:   "paid plan can read paid",
			status: "paid",
			viewer: ViewerAccess{IsAuthenticated: true, PlanCode: "BALANCE"},
			want:   true,
		},
		{
			name:   "admin can read preview",
			status: "preview",
			viewer: ViewerAccess{IsAuthenticated: true, IsAdmin: true},
			want:   true,
		},
		{
			name:   "non-admin cannot read preview",
			status: "preview",
			viewer: ViewerAccess{IsAuthenticated: true, PlanCode: "BALANCE"},
			want:   false,
		},
		{
			name:   "nobody can read draft",
			status: "draft",
			viewer: ViewerAccess{IsAuthenticated: true, IsAdmin: true},
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CanAccessArticle(tt.status, tt.viewer)
			if got != tt.want {
				t.Fatalf("CanAccessArticle(%q, %+v) = %v, want %v", tt.status, tt.viewer, got, tt.want)
			}
		})
	}
}

func TestAllowedStatusesForViewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		viewer ViewerAccess
		want   []string
	}{
		{
			name:   "anonymous",
			viewer: ViewerAccess{},
			want:   []string{"public"},
		},
		{
			name:   "authenticated start",
			viewer: ViewerAccess{IsAuthenticated: true, PlanCode: "START"},
			want:   []string{"public", "authorized"},
		},
		{
			name:   "authenticated paid",
			viewer: ViewerAccess{IsAuthenticated: true, PlanCode: "BALANCE"},
			want:   []string{"public", "authorized", "paid"},
		},
		{
			name:   "admin",
			viewer: ViewerAccess{IsAuthenticated: true, IsAdmin: true, PlanCode: "START"},
			want:   []string{"preview", "authorized", "paid", "public"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := allowedStatusesForViewer(tt.viewer)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("allowedStatusesForViewer(%+v) = %#v, want %#v", tt.viewer, got, tt.want)
			}
		})
	}
}

func TestIsPublishableStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status string
		want   bool
	}{
		{status: "draft", want: false},
		{status: "preview", want: false},
		{status: "authorized", want: true},
		{status: "paid", want: true},
		{status: "public", want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()
			got := isPublishableStatus(tc.status)
			if got != tc.want {
				t.Fatalf("isPublishableStatus(%q) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}
