package workspace

const (
	RoleOwner  = "owner"
	RoleMember = "member"
	RoleViewer = "viewer"
)

func CanManage(role string) bool {
	return role == RoleOwner
}

func InviteRoleOK(role string) bool {
	return role == RoleMember || role == RoleViewer
}
