package biz

import (
	"database/sql"
	"time"
)

// Paginator 是通用的分页结构体，可以嵌入到其他查询结构体中
type PaginatorWith[T any] struct {
	ID *int64 `query:"id" json:"id"`
	D  T
	PN int `query:"pn"` // Page number (0-indexed)
	PS int `query:"ps"` // Page size
}

// SetDefaults 设置分页默认值
func (p *PaginatorWith[T]) SetDefaults() {
	if p.PS <= 0 {
		p.PS = 10 // Default page size
	}
	if p.PN < 0 {
		p.PN = 0 // Default page number
	}
}

type Table[T any] struct {
	ID        *int64 `query:"id" json:"id"`
	D         T
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt sql.NullTime `json:"updated_at"`
}
