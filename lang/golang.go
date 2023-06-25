package lang

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"github.com/PackratPlus/UCD-builder/property"
)

func Go_PrintProps(packageName string, outDir string, props property.PropMap, tail string) {
	dirName := fmt.Sprintf("%s/%s", outDir, packageName)
	fileName := fmt.Sprintf("%s/%s.go", dirName, packageName)
	os.Mkdir(outDir, os.ModePerm)
	os.Mkdir(dirName, os.ModePerm)

	f, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	names := make([]string, 0)
	for nm := range props {
		names = append(names, nm)
	}
	sort.Strings(names)

	fmt.Fprintf(f, "package %s\n", packageName)
	fmt.Fprintf(f, "\nimport \"unicode\"\n")

	for _, nm := range names {
		go_PrintTo(props[nm], f)
	}

	fmt.Fprint(f, tail)
}

func go_PrintTo(bp *property.BinaryProperty, out io.Writer) {
	fmt.Fprintf(out, "\n// %s: %d codepoints\n", bp.Name, len(bp.CodePoints))

	rt := bp.ToRangeTable()

	fmt.Fprintf(out, "var %s = &unicode.RangeTable{\n", bp.Name)
	fmt.Fprintf(out, "  R16: []unicode.Range16{ /* %d */", len(rt.R16))
	for _, r := range rt.R16 {
		fmt.Fprintf(out, "\n    unicode.Range16{Lo: 0x%04X, Hi: 0x%04X, Stride: %d },",
			r.Lo, r.Hi, r.Stride)
	}
	if len(rt.R16) > 0 {
		fmt.Fprintf(out, "\n  ")
	}
	fmt.Fprintf(out, "},\n")
	fmt.Fprintf(out, "  R32: []unicode.Range32{ /* %d */", len(rt.R32))
	for _, r := range rt.R32 {
		fmt.Fprintf(out, "\n    unicode.Range32{Lo: 0x%06X, Hi: 0x%06X, Stride: %d },",
			r.Lo, r.Hi, r.Stride)
	}
	if len(rt.R32) > 0 {
		fmt.Fprintf(out, "\n  ")
	}
	fmt.Fprintf(out, "},\n")
	fmt.Fprintf(out, "  LatinOffset: %d,\n", rt.LatinOffset)
	fmt.Fprintf(out, "}\n")
}
