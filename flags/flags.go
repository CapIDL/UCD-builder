package flags

import (
	flag "github.com/spf13/pflag"
)

var DataPath string
var OutDir string
var Lang string

func ProcessFlags() {
	flag.StringVarP(&DataPath, "data", "d", "", "path to unicode data tree for desired version")
	flag.StringVarP(&OutDir, "out", "o", "out/", "path for output files")
	flag.StringVarP(&Lang, "lamg", "l", "go", "target language name")
	flag.Parse()
}

func Args() []string {
	return flag.Args()
}

func Arg(n int) string {
	return flag.Arg(n)
}
