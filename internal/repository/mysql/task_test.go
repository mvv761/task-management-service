package mysql

import (
	"encoding/json"
	"testing"
	"time"

	"task-management-service/internal/domain"
)

func TestBuildTaskChanges_TracksChangedFieldsForHistory(t *testing.T) {
	oldDescription := "old description"
	newDescription := "new description"
	oldAssignee := uint64(2)
	newAssignee := uint64(3)
	oldClosedAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	newClosedAt := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)

	current := &domain.Task{
		ID:          10,
		Title:       "old title",
		Description: &oldDescription,
		Status:      domain.TaskStatusTodo,
		AssigneeID:  &oldAssignee,
		ClosedAt:    &oldClosedAt,
	}
	updated := &domain.Task{
		ID:          10,
		Title:       "new title",
		Description: &newDescription,
		Status:      domain.TaskStatusDone,
		AssigneeID:  &newAssignee,
		ClosedAt:    &newClosedAt,
	}

	changes := buildTaskChanges(current, updated)
	for _, field := range []string{"title", "description", "status", "assignee_id", "closed_at"} {
		if _, ok := changes[field]; !ok {
			t.Fatalf("expected change for field %q in %#v", field, changes)
		}
	}

	data, err := json.Marshal(changes)
	if err != nil {
		t.Fatalf("changes must be valid JSON: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("expected valid JSON changes, got %s", data)
	}
}

func TestBuildTaskChanges_IgnoresUnchangedFields(t *testing.T) {
	description := "description"
	assignee := uint64(2)
	closedAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	current := &domain.Task{
		ID:          10,
		Title:       "title",
		Description: &description,
		Status:      domain.TaskStatusTodo,
		AssigneeID:  &assignee,
		ClosedAt:    &closedAt,
	}
	updated := &domain.Task{
		ID:          10,
		Title:       "title",
		Description: &description,
		Status:      domain.TaskStatusTodo,
		AssigneeID:  &assignee,
		ClosedAt:    &closedAt,
	}

	if changes := buildTaskChanges(current, updated); len(changes) != 0 {
		t.Fatalf("expected no changes, got %#v", changes)
	}
}
