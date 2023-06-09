// Utility to build unicode property RangeTables directly from the UCD
// database.
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/CapIDL/UCD-builder/flags"
	"github.com/CapIDL/UCD-builder/lang"
	"github.com/CapIDL/UCD-builder/property"
)

// From table 12 of UAX44
type CatValue struct {
	ShortName string
	LongName  string
	Merges    []string
}

// Converted from PropValueAliases.txt per UAX #44
var CatMap = []CatValue{
	{"C", "Other", []string{"Cc", "Cf", "Cn", "Co", "Cs"}},
	{"Cc", "cntrl", nil},
	{"Cc", "Control", nil},
	{"Cf", "Format", nil},
	{"Cn", "Unassigned", nil},
	{"Co", "Private_Use", nil},
	{"Cs", "Surrogate", nil},
	{"L", "Letter", []string{"Ll", "Lm", "Lo", "Lt", "Lu"}},
	{"LC", "Cased_Letter", []string{"Ll", "Lt", "Lu"}},
	{"Ll", "Lowercase_Letter", nil},
	{"Lm", "Modifier_Letter", nil},
	{"Lo", "Other_Letter", nil},
	{"Lt", "Titlecase_Letter", nil},
	{"Lu", "Uppercase_Letter", nil},
	{"M", "Combining_Mark", nil},
	{"M", "Mark", []string{"Mc", "Me", "Mn"}},
	{"Mc", "Spacing_Mark", nil},
	{"Me", "Enclosing_Mark", nil},
	{"Mn", "Nonspacing_Mark", nil},
	{"N", "Number", []string{"Nd", "Nl", "No"}},
	{"Nd", "digit", nil},
	{"Nd", "Decimal_Number", nil},
	{"Nl", "Letter_Number", nil},
	{"No", "Other_Number", nil},
	{"P", "Punctuation", []string{"Pc", "Pd", "Pe", "Pf", "Pi", "Po", "Ps"}},
	{"P", "punct", nil},
	{"Pc", "Connector_Punctuation", nil},
	{"Pd", "Dash_Punctuation", nil},
	{"Pe", "Close_Punctuation", nil},
	{"Pf", "Final_Punctuation", nil},
	{"Pi", "Initial_Punctuation", nil},
	{"Po", "Other_Punctuation", nil},
	{"Ps", "Open_Punctuation", nil},
	{"S", "Symbol", []string{"Sc", "Sk", "Sm", "So"}},
	{"Sc", "Currency_Symbol", nil},
	{"Sk", "Modifier_Symbol", nil},
	{"Sm", "Math_Symbol", nil},
	{"So", "Other_Symbol", nil},
	{"Z", "Separator", []string{"Zl", "Zp", "Zs"}},
	{"Zl", "Line_Separator", nil},
	{"Zp", "Paragraph_Separator", nil},
	{"Zs", "Space_Separator", nil},
}

func main() {
	flags.ProcessFlags()

	if len(flags.Args()) != 2 {
		log.Fatalf("Usage: digest <unicode version> <output directory> (%d)\n", len(flags.Args()))
	}

	version := flags.Arg(0)
	matched, err := regexp.MatchString("[0-9]{1,2}(\\.[0-9]){2}", version)

	if err != nil || !matched {
		log.Fatalf("Unicode version number should be major.minor.update\n")
	}

	outDir := flags.Arg(1)

	langName := flags.Lang
	lang, ok := lang.Language[langName]
	if !ok {
		log.Fatalf("Unknown or unsupported output language \"%s\"\n", langName)
	}

	// ------------------------------------------------------------------------
	// Scripts
	// ------------------------------------------------------------------------
	scriptsFile := "ucd/Scripts.txt"
	ucdFile := "ucd/UnicodeData.txt"
	propsFile := "ucd/PropList.txt"
	derivedPropsFile := "ucd/DerivedCoreProperties.txt"
	emojiPropsFile := "ucd/emoji/emoji-data.txt"

	// if true {
	// 	names := property.FetchPropertyNames(scriptsFile, 1)
	// 	fmt.Printf("Scripts:")

	// 	for i, nm := range names {
	// 		if i%8 == 0 {
	// 			fmt.Printf("\n  ")
	// 		}
	// 		fmt.Printf("%s, ", nm)
	// 	}

	// 	fmt.Printf("\n\n")
	// }

	var props map[string]*property.BinaryProperty

	// ------------------------------------------------------------------------
	// Scripts
	// ------------------------------------------------------------------------
	props, _ = property.ParsePropertyFile(version, scriptsFile, 1)
	lang.PrintProps("script", outDir, props, "")
	props = nil

	// ------------------------------------------------------------------------
	// Categories
	// ------------------------------------------------------------------------
	props, _ = property.ParsePropertyFile(version, ucdFile, 2)

	// Need to derive the "Other, not assigned" category. Gather everything that
	// *is* assigned and negate that:
	allProps := make([](*property.BinaryProperty), 0)
	for _, v := range props {
		allProps = append(allProps, v)
	}
	assigned := property.Merge("assigned", allProps...)
	unassigned := &property.BinaryProperty{
		Name:       "Cn",
		CodePoints: make(map[rune]bool),
	}
	for rn := rune(0); rn <= 0x10FFFF; rn++ {
		if _, ok := assigned.CodePoints[rn]; !ok {
			unassigned.CodePoints[rn] = true
		}
	}
	props["Cn"] = unassigned

	// We now have all of the catagories whose name takes the form Xx. Gather
	// together the coalesced general categories:
	for _, cat := range CatMap {
		if _, ok := props[cat.ShortName]; ok {
			continue
		}

		subProps := make([](*property.BinaryProperty), 0)

		for _, name := range cat.Merges {
			subProp, ok := props[name]
			if !ok {
				panic(fmt.Sprintf("Constituent prop %s not found!\n", name))
			}
			subProps = append(subProps, subProp)
		}
		props[cat.ShortName] = property.Merge(cat.ShortName, subProps...)
	}

	tail := "\n// Long names:\n\n"
	for _, cat := range CatMap {
		tail = tail + fmt.Sprintf("var %s = %s\n", cat.LongName, cat.ShortName)
	}
	tail = tail + "\n"

	lang.PrintProps("category", outDir, props, tail)

	props = nil

	// props, err = property.ParseCategories(ucdFile)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Category parse failed with error %s\n", err.Error())
	// 	os.Exit(-1)
	// }

	// PrintProps("category", props)
	// props = nil

	// ------------------------------------------------------------------------
	// Properties
	// ------------------------------------------------------------------------
	props, _ = property.ParsePropertyFile(version, propsFile, 1)
	cumProps := props
	props, _ = property.ParsePropertyFile(version, derivedPropsFile, 1)

	for k, v := range props {
		cumProps[k] = v
	}
	props = nil

	props, _ = property.ParsePropertyFile(version, emojiPropsFile, 1)

	for k, v := range props {
		cumProps[k] = v
	}
	props = nil

	lang.PrintProps("property", outDir, cumProps, "")
	props = nil

	os.Exit(0)
}
