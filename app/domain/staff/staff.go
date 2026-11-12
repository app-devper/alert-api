package staff

import (
	"alert/app/data/entities"
	"alert/app/data/repositories"

	"github.com/app-devper/um-api/sessionclient"
	"go.mongodb.org/mongo-driver/bson"
)

// Staff is the person behind a staff request, resolved once by RequireBranch:
// who they are (UM), the Branch their Alert Permission names, and what that
// Permission lets them trigger. Route-level Role checks stay in UM's hands.
type Staff struct {
	UserId string
	Role   sessionclient.Role
	Scope  repositories.Scope

	allowedEventTypes []string
}

// FromPermission builds the Staff for a User holding the given Alert Permission.
func FromPermission(userId string, role sessionclient.Role, permission entities.StaffPermission) Staff {
	return Staff{
		UserId:            userId,
		Role:              role,
		Scope:             repositories.Scope{ClientId: permission.ClientId, BranchId: permission.BranchId},
		allowedEventTypes: append([]string(nil), permission.AllowedEventTypes...),
	}
}

// MayTrigger reports whether Staff may trigger eventType: a MANAGER or above
// may trigger any, anyone else only the types their Permission lists.
func (s Staff) MayTrigger(eventType string) bool {
	if s.isManager() {
		return true
	}
	return entities.StaffPermission{AllowedEventTypes: s.allowedEventTypes}.AllowsEventType(eventType)
}

// MayTest reports whether Staff may send test alerts.
func (s Staff) MayTest() bool { return s.isManager() }

// MayOverrideCooldown reports whether Staff may trigger inside a Cooldown.
func (s Staff) MayOverrideCooldown() bool { return s.isManager() }

// AuditEntry returns an audit log entry with Client, Branch and Actor filled from s.
func (s Staff) AuditEntry(action string, result string, detail bson.M) entities.AuditLog {
	return entities.AuditLog{
		ClientId: s.Scope.ClientId,
		BranchId: s.Scope.BranchId,
		Actor:    s.UserId,
		Action:   action,
		Detail:   detail,
		Result:   result,
	}
}

func (s Staff) isManager() bool { return s.Role.AtLeast(sessionclient.RoleManager) }
