package util

const (
	DepositorRole = "depositor"
	BankerRole    = "banker"
)

var rolePermissions = map[string]map[string]bool{
	"banker": {
		"users:update":     true,
		"accounts:create":  true,
		"accounts:read":    true,
		"accounts:list":    true,
		"transfers:create": true,
	},
	"depositor": {
		"users:update":     true,
		"accounts:read":    true,
		"transfers:create": true,
	},
}

func HasPermission(role, perm string) bool {
	return rolePermissions[role][perm]
}
