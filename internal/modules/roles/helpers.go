package roles

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
)

func mapRole(m domain.Role) view { return view{ID: m.ID.ID, Name: m.Name} }
