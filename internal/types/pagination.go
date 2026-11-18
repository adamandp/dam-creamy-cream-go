package types

type PaginationQuery struct {
	Page  int `query:"page,default:1" validate:"min=1,max=100"`
	Limit int `query:"limit,default:10" validate:"min=1,max=100"`
}
