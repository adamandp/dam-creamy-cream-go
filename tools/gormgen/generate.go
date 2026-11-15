package main

import (
	"dam-creamy-cream-go/internal/dal/model"

	"gorm.io/gen"
)

func main() {
	// Generate typed query APIs from manually maintained Go structs.
	// No database connection or schema introspection is required.
	g := gen.NewGenerator(gen.Config{
		OutPath: "internal/dal/query",
	})

	g.ApplyBasic(model.All()...)
	g.Execute()
}
