package codegen

import (
	"fmt"
	"strings"
	"unicode"

	"goa.design/goa.v2/design"
)

var (
	// common words who need to keep their
	commonInitialisms = map[string]bool{
		"API":   true,
		"ASCII": true,
		"CPU":   true,
		"CSS":   true,
		"DNS":   true,
		"EOF":   true,
		"GUID":  true,
		"HTML":  true,
		"HTTP":  true,
		"HTTPS": true,
		"ID":    true,
		"IP":    true,
		"JMES":  true,
		"JSON":  true,
		"JWT":   true,
		"LHS":   true,
		"OK":    true,
		"QPS":   true,
		"RAM":   true,
		"RHS":   true,
		"RPC":   true,
		"SLA":   true,
		"SMTP":  true,
		"SQL":   true,
		"SSH":   true,
		"TCP":   true,
		"TLS":   true,
		"TTL":   true,
		"UDP":   true,
		"UI":    true,
		"UID":   true,
		"UUID":  true,
		"URI":   true,
		"URL":   true,
		"UTF8":  true,
		"VM":    true,
		"XML":   true,
		"XSRF":  true,
		"XSS":   true,
	}

	// reserved golang keywords and package names
	reserved = map[string]bool{
		"byte":       true,
		"complex128": true,
		"complex64":  true,
		"float32":    true,
		"float64":    true,
		"int":        true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"int8":       true,
		"rune":       true,
		"string":     true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uint8":      true,

		// reserved keywords
		"break":       true,
		"case":        true,
		"chan":        true,
		"const":       true,
		"continue":    true,
		"default":     true,
		"defer":       true,
		"else":        true,
		"fallthrough": true,
		"for":         true,
		"func":        true,
		"go":          true,
		"goto":        true,
		"if":          true,
		"import":      true,
		"interface":   true,
		"map":         true,
		"package":     true,
		"range":       true,
		"return":      true,
		"select":      true,
		"struct":      true,
		"switch":      true,
		"type":        true,
		"var":         true,

		// stdlib and goa packages used by generated code
		"fmt":  true,
		"http": true,
		"json": true,
		"os":   true,
		"url":  true,
		"time": true,
	}
)

// GoTypeRef returns the Go code that refers to the Go type which matches the given data type
func GoTypeRef(dt design.DataType) string {
	tname := GoTypeName(dt)
	if design.IsObject(dt) {
		return "*" + tname
	}
	return tname
}

// GoTypeName returns the Go type name for a data type.
// todo: TBD add support for maps, objects and usertypes
func GoTypeName(dt design.DataType) string {
	switch actual := dt.(type) {
	case design.Primitive:
		return GoNativeType(dt)
	case *design.Array:
		return "[]" + GoTypeRef(actual.ElemType.Type)
	default:
		panic(fmt.Sprintf("goa bug: unknown type %#v", actual))
	}
}

// GoNativeType returns the Go built-in type from which instances of provided datatype can be initialized.
// todo: TBD add support for maps, objects and usertypes
func GoNativeType(t design.DataType) string {
	switch actual := t.(type) {
	case design.Primitive:
		switch actual.Kind() {
		case design.BooleanKind:
			return "bool"
		case design.Int32Kind:
			return "int32"
		case design.Int64Kind:
			return "int64"
		case design.UInt32Kind:
			return "uint32"
		case design.UInt64Kind:
			return "uint64"
		case design.Float32Kind:
			return "float32"
		case design.Float64Kind:
			return "float64"
		case design.StringKind:
			return "string"
		case design.AnyKind:
			return "interface{}"
		default:
			panic(fmt.Sprintf("goa bug: unknown primitive type %#v", actual))
		}
	case *design.Array:
		return "[]" + GoNativeType(actual.ElemType.Type)
	case design.CompositeExpr:
		return GoNativeType(actual.Attribute().Type)
	default:
		panic(fmt.Sprintf("goa bug: unknown type %#v", actual))
	}
}

// Goify makes a valid Go identifier out of any string.
// It does that by removing any non letter and non digit character and by making sure the first
// character is a letter or "_".
// Goify produces a "CamelCase" version of the string, if firstUpper is true the first character
// of the identifier is uppercase otherwise it's lowercase.
func Goify(str string, firstUpper bool) string {
	runes := []rune(str)

	// remove trailing invalid identifiers (makes code below simpler)
	runes = removeTrailingInvalid(runes)

	w, i := 0, 0 // index of start of word, scan
	for i+1 <= len(runes) {
		eow := false // whether we hit the end of a word

		// remove leading invalid identifiers
		runes = removeInvalidAtIndex(i, runes)

		if i+1 == len(runes) {
			eow = true
		} else if !validIdentifier(runes[i]) {
			// get rid of it
			runes = append(runes[:i], runes[i+1:]...)
		} else if runes[i+1] == '_' {
			// underscore; shift the remainder forward over any run of underscores
			eow = true
			n := 1
			for i+n+1 < len(runes) && runes[i+n+1] == '_' {
				n++
			}
			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		} else if unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]) {
			// lower->non-lower
			eow = true
		}
		i++
		if !eow {
			continue
		}

		// [w,i] is a word.
		word := string(runes[w:i])
		// is it one of our initialisms?
		if u := strings.ToUpper(word); commonInitialisms[u] {
			if firstUpper {
				u = strings.ToUpper(u)
			} else if w == 0 {
				u = strings.ToLower(u)
			}

			// All the common initialisms are ASCII,
			// so we can replace the bytes exactly.
			copy(runes[w:], []rune(u))
		} else if w > 0 && strings.ToLower(word) == word {
			// already all lowercase, and not the first word, so uppercase the first character.
			runes[w] = unicode.ToUpper(runes[w])
		} else if w == 0 && strings.ToLower(word) == word && firstUpper {
			runes[w] = unicode.ToUpper(runes[w])
		}
		if w == 0 && !firstUpper {
			runes[w] = unicode.ToLower(runes[w])
		}
		//advance to next word
		w = i
	}

	return fixReserved(string(runes))
}

// validIdentifier returns true if the rune is a letter or number
func validIdentifier(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// fixReserved appends an underscore on to Go reserved keywords.
func fixReserved(w string) string {
	if reserved[w] {
		w += "_"
	}
	return w
}

// removeTrailingInvalid removes trailing invalid identifiers from runes.
func removeTrailingInvalid(runes []rune) []rune {
	valid := len(runes) - 1
	for ; valid >= 0 && !validIdentifier(runes[valid]); valid-- {
	}

	return runes[0: valid+1]
}

// removeInvalidAtIndex removes consecutive invalid identifiers from runes starting at index i.
func removeInvalidAtIndex(i int, runes []rune) []rune {
	valid := i
	for ; valid < len(runes) && !validIdentifier(runes[valid]); valid++ {
	}

	return append(runes[:i], runes[valid:]...)
}
