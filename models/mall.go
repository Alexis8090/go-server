package models

type Mall struct {
	Name     *string `json:"name" validate:"required"`
	Location *string `json:"location" validate:"required"`
}
