package server

type LoginInput struct {
	Username   string `json:"username,omitempty"`
	AssignedId string `json:"assigned_id,omitempty"`
}
