package policy

import (
	"errors"
	"testing"

	"task-management-service/internal/domain"
)

func TestCanEditTask_RoleBasedPolicy(t *testing.T) {
	assigneeID := uint64(42)
	title := "updated title"
	status := domain.TaskStatusDone

	task := domain.Task{
		ID:         1,
		TeamID:     10,
		Title:      "initial title",
		Status:     domain.TaskStatusTodo,
		CreatedBy:  7,
		AssigneeID: &assigneeID,
		Version:    1,
	}

	tests := []struct {
		name    string
		actor   domain.TeamMember
		patch   domain.TaskPatch
		wantErr error
	}{
		{
			name:  "owner can edit any field",
			actor: domain.TeamMember{TeamID: task.TeamID, UserID: 1, Role: domain.TeamRoleOwner},
			patch: domain.TaskPatch{Title: &title, Version: 1},
		},
		{
			name:  "admin can edit any field",
			actor: domain.TeamMember{TeamID: task.TeamID, UserID: 2, Role: domain.TeamRoleAdmin},
			patch: domain.TaskPatch{Title: &title, Version: 1},
		},
		{
			name:  "creator member can edit any field",
			actor: domain.TeamMember{TeamID: task.TeamID, UserID: task.CreatedBy, Role: domain.TeamRoleMember},
			patch: domain.TaskPatch{Title: &title, Version: 1},
		},
		{
			name:  "assignee member can change status",
			actor: domain.TeamMember{TeamID: task.TeamID, UserID: assigneeID, Role: domain.TeamRoleMember},
			patch: domain.TaskPatch{Status: &status, Version: 1},
		},
		{
			name:    "assignee member cannot edit title",
			actor:   domain.TeamMember{TeamID: task.TeamID, UserID: assigneeID, Role: domain.TeamRoleMember},
			patch:   domain.TaskPatch{Title: &title, Version: 1},
			wantErr: domain.ErrInsufficientPermissions,
		},
		{
			name:    "unrelated member cannot edit task",
			actor:   domain.TeamMember{TeamID: task.TeamID, UserID: 99, Role: domain.TeamRoleMember},
			patch:   domain.TaskPatch{Status: &status, Version: 1},
			wantErr: domain.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanEditTask(tt.actor, task, tt.patch)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
