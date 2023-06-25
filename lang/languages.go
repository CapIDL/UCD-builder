package lang

import "github.com/PackratPlus/UCD-builder/property"

type Lang struct {
	Name string

	PrintProps func(packageName string, outDir string, props property.PropMap, tail string)
}

var Language = make(map[string](*Lang))

func init() {
	Language["go"] = &Lang{
		"go",
		Go_PrintProps,
	}
}
