package property

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/exp/constraints"

	"github.com/PackratPlus/UCD-builder/flags"
)

type BinaryProperty struct {
	Name string
	// Min        rune
	// Max        rune
	// SigBits    int
	CodePoints map[rune]bool
}

func Merge(name string, bps ...*BinaryProperty) *BinaryProperty {
	out := &BinaryProperty{
		Name:       name,
		CodePoints: make(map[rune]bool),
	}

	for _, bp := range bps {
		for k := range bp.CodePoints {
			out.CodePoints[k] = true
		}
	}

	return out
}

func (bp *BinaryProperty) AddCodePoint(r rune) {
	bp.CodePoints[r] = true
}

func (bp *BinaryProperty) ToRangeTable() *unicode.RangeTable {
	var LatinOffset int

	rl16 := []unicode.Range16{}
	rl32 := []unicode.Range32{}

	var r16 unicode.Range16
	var r32 unicode.Range32

	codePoints := make([]rune, 0, len(bp.CodePoints))
	for cp := range bp.CodePoints {
		codePoints = append(codePoints, cp)
	}
	sort.Slice(codePoints, func(i, j int) bool {
		return codePoints[i] < codePoints[j]
	})

	for _, cp := range codePoints {
		if cp <= 0xffff {
			ucp := uint16(cp)

			if r16.Stride == 0 {
				r16 = unicode.Range16{Lo: ucp, Hi: ucp, Stride: 1}
			} else if r16.Hi <= unicode.MaxLatin1 && ucp > unicode.MaxLatin1 {
				// Special case: range break at MaxLatin1 to speed ascii and latin1
				// search.
				rl16 = append(rl16, r16)
				if r16.Hi <= unicode.MaxLatin1 {
					LatinOffset++
				}
				r16 = unicode.Range16{Lo: ucp, Hi: ucp, Stride: 1}
			} else if ucp == r16.Hi+r16.Stride {
				r16.Hi = ucp
			} else if useStride && r16.Lo == r16.Hi {
				r16.Stride = ucp - r16.Lo
				r16.Hi = ucp
			} else {
				rl16 = append(rl16, r16)
				if r16.Hi <= unicode.MaxLatin1 {
					LatinOffset++
				}
				r16 = unicode.Range16{Lo: ucp, Hi: ucp, Stride: 1}
			}
		} else {
			ucp := uint32(cp)

			if r32.Stride == 0 {
				r32 = unicode.Range32{Lo: ucp, Hi: ucp, Stride: 1}
			} else if ucp == r32.Hi+r32.Stride {
				r32.Hi = ucp
			} else if useStride && r32.Lo == r32.Hi {
				r32.Stride = ucp - r32.Lo
				r32.Hi = ucp
			} else {
				rl32 = append(rl32, r32)
				r32 = unicode.Range32{Lo: ucp, Hi: ucp, Stride: 1}
			}
		}
	}

	if r16.Hi > 0 {
		rl16 = append(rl16, r16)
		if r16.Hi <= unicode.MaxLatin1 {
			LatinOffset++
		}
	}
	if r32.Hi > 0 {
		rl32 = append(rl32, r32)
	}

	return &unicode.RangeTable{
		R16:         rl16,
		R32:         rl32,
		LatinOffset: LatinOffset,
	}
}

const useStride = true

type PropMap = map[string](*BinaryProperty)

type OrderedAndComparable interface {
	constraints.Ordered
	comparable
}

func MapKeys[K OrderedAndComparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func FetchPropertyNames(filePath string, nameCol int) []string {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Script name parse failed with error %s\n", err.Error())
		os.Exit(-1)
	}
	defer file.Close()

	propNames := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := strings.Split(line, ";")
		if len(parts) <= nameCol {
			fmt.Fprintf(os.Stderr, "Name column number %d exceeds total column count %d\n",
				nameCol, len(parts))
			os.Exit(-1)
		}

		// Generic name extraction - unless we are looking for the code point name,
		// this works for both UnicodeData and also the newer files. Need to remove
		// trailing comment (if any) and trim white space.
		propName := strings.Split(parts[nameCol], "#")[0]
		propName = strings.TrimSpace(propName)

		propNames[propName] = true
	}

	return MapKeys(propNames)
}

func ParsePropertyFile(version string, filePath string, nameCol int) (PropMap, error) {
	dataPath := fmt.Sprintf("%s/%s", flags.DataPath, version)
	path := fmt.Sprintf("%s/%s", dataPath, filePath)
	url := fmt.Sprintf("http://unicode.org/Public/%s/%s", version, filePath)

	if flags.DataPath != "" {
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()

			log.Printf("Using %s\n", path)
			scanner := bufio.NewScanner(file)

			pmap, err := ParseProperties(scanner, nameCol)
			if err == nil {
				return pmap, err
			}
		} else {
			fmt.Fprintf(os.Stderr, "Unable to open UCD file \"%s\"\n", path)
			// Fall through and fetch
		}
	}

	// Couldn't find the file locally - grab it from upstream
	log.Printf("Fetching %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode > 299 {
		log.Fatalf("Get failed with setatus code: %d. Is this a valid version?\n", resp.StatusCode)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	return ParseProperties(scanner, nameCol)
}

func ParseProperties(scanner *bufio.Scanner, nameCol int) (PropMap, error) {
	properties := make(PropMap)

	var err error
	var start, end, first int64

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := strings.Split(line, ";")
		if len(parts) <= nameCol {
			fmt.Fprintf(os.Stderr, "Name column number %d exceeds total column count %d\n",
				nameCol, len(parts))
			os.Exit(-1)
		}

		codePointRange := strings.TrimSpace(parts[0])

		// Second column is always present and usually some form of col2:
		col2 := strings.Split(parts[1], "#")[0]
		col2 = strings.TrimSpace(col2)

		propName := strings.Split(parts[nameCol], "#")[0]
		propName = strings.TrimSpace(propName)

		property, ok := properties[propName]
		if !ok {
			property = &BinaryProperty{
				Name:       propName,
				CodePoints: make(map[rune]bool),
			}
			properties[propName] = property
		}

		rangeParts := strings.Split(codePointRange, "..")
		start, err = strconv.ParseInt(rangeParts[0], 16, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Malformed code point range \"%s\" (%s, %d) in first column\n", codePointRange, rangeParts[0], len(rangeParts))
			os.Exit(-1)
		}

		end = start

		if len(rangeParts) == 2 {
			end, err = strconv.ParseInt(rangeParts[1], 16, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Malformed code point range \"%s\" (%s, %d) in first column\n", codePointRange, rangeParts[1], len(rangeParts))
				os.Exit(-1)
			}
		} else if col2[0] == '<' && strings.Contains(col2, ",") {
			// In UnicodeData.txt, ranges are indicated by a bodge in the second column.
			// This is the *only* case in the entire data set where the first character
			// in the second column can be '<'.

			col2 = col2[1 : len(col2)-1]
			col2 = strings.Split(col2, ",")[1]
			col2 = strings.TrimSpace(col2)

			// If col2 is now "First", then this line gave the first codepoint in a
			// range. Next line will have "Last" in the same position, and will give
			// us the last code point.
			if col2 == "First" {
				first = start
				continue
			} else {
				start = first
				end, err = strconv.ParseInt(rangeParts[0], 16, 32)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Malformed code point range \"%s\" (%s, %d) in first column\n", codePointRange, rangeParts[0], len(rangeParts))
					os.Exit(-1)
				}
			}
		}

		for i := start; i <= end; i++ {
			property.AddCodePoint(rune(i))
		}
	}

	return properties, nil
}
