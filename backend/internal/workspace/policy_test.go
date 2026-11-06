package workspace

import "testing"

func TestCanManage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want bool
	}{
		{RoleOwner, true},
		{RoleMember, false},
		{RoleViewer, false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			t.Parallel()
			if got := CanManage(tt.role); got != tt.want {
				t.Fatalf("CanManage(%q)=%v want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestInviteRoleOK(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want bool
	}{
		{RoleMember, true},
		{RoleViewer, true},
		{RoleOwner, false},
		{"admin", false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			t.Parallel()
			if got := InviteRoleOK(tt.role); got != tt.want {
				t.Fatalf("InviteRoleOK(%q)=%v want %v", tt.role, got, tt.want)
			}
		})
	}
}
