package policy

import "task-management-service/internal/domain"

func CanEditTask(actor domain.TeamMember, task domain.Task, patch domain.TaskPatch) error {
	if actor.Role == domain.TeamRoleOwner || actor.Role == domain.TeamRoleAdmin {
		return nil
	}

	if actor.UserID == task.CreatedBy {
		return nil
	}

	if task.AssigneeID != nil &&
		actor.UserID == *task.AssigneeID {

		if patch.Title != nil ||
			patch.DescriptionSet ||
			patch.AssigneeSet {
			return domain.ErrInsufficientPermissions
		}

		if patch.Status != nil {
			return nil
		}
	}

	return domain.ErrInsufficientPermissions
}
