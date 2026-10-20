package dto

type CreateTeamRequest struct {
	Name string `json:"name" binding:"required" example:"Platform"`
}

type TeamResponse struct {
	ID        uint64 `json:"id" example:"1"`
	Name      string `json:"name" example:"Platform"`
	CreatedBy uint64 `json:"created_by" example:"1"`
}

type ListTeamsResponse struct {
	Teams []TeamResponse `json:"teams"`
}

type InviteMemberRequest struct {
	UserID uint64 `json:"user_id" binding:"required" example:"2"`
	Role   string `json:"role" binding:"required" enums:"admin,member" example:"member"`
}

type UpdateTeamMemberRoleRequest struct {
	Role string `json:"role" binding:"required" enums:"admin,member" example:"admin"`
}
