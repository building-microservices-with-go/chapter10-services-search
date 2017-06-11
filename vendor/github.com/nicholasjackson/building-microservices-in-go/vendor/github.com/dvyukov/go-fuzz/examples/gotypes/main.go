// Copyright 2015 Dmitry Vyukov. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package gotypes

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// https://github.com/golang/go/issues/11327
var bigNum = regexp.MustCompile("(\\.[0-9]*)|([0-9]+)[eE]\\-?\\+?[0-9]{3,}")
var bigNum2 = regexp.MustCompile("[0-9]+[pP][0-9]{3,}") // see issue 11364

// https://github.com/golang/go/issues/11274
var formatBug1 = regexp.MustCompile("\\*/[ \t\n\r\f\v]*;")
var formatBug2 = regexp.MustCompile(";[ \t\n\r\f\v]*/\\*")

var issue11590 = regexp.MustCompile(": cannot convert .* \\(untyped int constant .*\\) to complex")
var issue11590_2 = regexp.MustCompile(": [0-9]+ (untyped int constant) overflows complex")
var issue11370 = regexp.MustCompile("\\\"[ \t\n\r\f\v]*\\[")

var fpRounding = regexp.MustCompile(" \\(untyped float constant .*\\) truncated to ")
var something = regexp.MustCompile(" constant .* overflows ")

var gcCrash = regexp.MustCompile("\n/tmp/fuzz\\.gc[0-9]+:[0-9]+: internal compiler error: ")
var asanCrash = regexp.MustCompile("\n==[0-9]+==ERROR: AddressSanitizer: ")

const (
	testGccgo = false
)

func Fuzz(data []byte) int {
	if bigNum.Match(data) || bigNum2.Match(data) {
		return -1
	}
	goErr := gotypes(data)
	gcErr := gc(data)
	gccgoErr := gcErr
	if testGccgo {
		gccgoErr = gccgo(data)
	}

	if goErr == nil && gcErr != nil {
		if strings.Contains(gcErr.Error(), "line number out of range") {
			// https://github.com/golang/go/issues/11329
			return -1
		}
		if strings.Contains(gcErr.Error(), "larger than address space") {
			// Gc is more picky at rejecting huge objects.
			return -1
		}
		if strings.Contains(gcErr.Error(), "non-canonical import path") {
			return -1
		}
		if strings.Contains(gcErr.Error(), "constant shift overflow") {
			// ???
			return -1
		}
		if something.MatchString(gcErr.Error()) {
			// ???
			return -1
		}
	}

	if gcErr == nil && goErr != nil {
		if issue11370.MatchString(goErr.Error()) {
			return -1
		}
	}

	if gccgoErr == nil && goErr != nil {
		if strings.Contains(goErr.Error(), "invalid operation: stupid shift count") {
			// https://github.com/golang/go/issues/11524
			return -1
		}
		if fpRounding.MatchString(goErr.Error()) {
			// gccgo has different rounding
			return -1
		}
		if strings.Contains(goErr.Error(), "illegal byte order mark") {
			// on "package\rG\n//line \ufeff:1" input, not filed.
			return -1
		}
	}

	if goErr == nil && gccgoErr != nil {
		if bytes.Contains(data, []byte("0i")) &&
			(strings.Contains(gccgoErr.Error(), "incompatible types in binary expression") ||
				strings.Contains(gccgoErr.Error(), "initialization expression has wrong type")) {
			// https://github.com/golang/go/issues/11564
			// https://github.com/golang/go/issues/11563
			return -1
		}
	}

	// go-fuzz is too smart so it can generate a program that contains "internal compiler error" in an error message :)
	if gcErr != nil && (gcCrash.MatchString(gcErr.Error()) ||
		strings.Contains(gcErr.Error(), "\nruntime error: ") ||
		strings.HasPrefix(gcErr.Error(), "runtime error: ") ||
		strings.Contains(gcErr.Error(), "%!")) { // bad format string
		fmt.Printf("gc result: %v\n", gcErr)
		panic("gc compiler crashed")
	}

	const gccgoCrash = "go1: internal compiler error:"
	if gccgoErr != nil && (strings.HasPrefix(gccgoErr.Error(), gccgoCrash) || strings.Contains(gccgoErr.Error(), "\n"+gccgoCrash)) {
		if strings.Contains(gccgoErr.Error(), "go1: internal compiler error: in do_export, at go/gofrontend/types.cc") {
			// https://github.com/golang/go/issues/12321
			return -1
		}
		if strings.Contains(gccgoErr.Error(), "go1: internal compiler error: in do_lower, at go/gofrontend/expressions.cc") {
			// https://github.com/golang/go/issues/12615
			return -1
		}
		if strings.Contains(gccgoErr.Error(), "go1: internal compiler error: in wide_int_to_tree, at tree.c") {
			// https://github.com/golang/go/issues/12618
			return -1
		}
		if strings.Contains(gccgoErr.Error(), "go1: internal compiler error: in uniform_vector_p, at tree.c") {
			// https://github.com/golang/go/issues/12935
			return -1
		}
		if strings.Contains(gccgoErr.Error(), "go1: internal compiler error: in do_determine_type, at go/gofrontend/expressions.h") {
			// https://github.com/golang/go/issues/12937
			return -1
		}
		if strings.Contains(gccgoErr.Error(), "go1: internal compiler error: in do_get_backend, at go/gofrontend/expressions.cc") {
			// https://github.com/golang/go/issues/12939
			return -1
		}
		fmt.Printf("gccgo result: %v\n", gccgoErr)
		panic("gccgo compiler crashed")
	}

	if gccgoErr != nil && asanCrash.MatchString(gccgoErr.Error()) {
		fmt.Printf("gccgo result: %v\n", gccgoErr)
		panic("gccgo compiler crashed")
	}

	if (goErr == nil) != (gcErr == nil) || (goErr == nil) != (gccgoErr == nil) {
		fmt.Printf("go/types result: %v\n", goErr)
		fmt.Printf("gc result: %v\n", gcErr)
		fmt.Printf("gccgo result: %v\n", gccgoErr)
		panic("gc, gccgo and go/types disagree")
	}
	if goErr != nil {
		return 0

	}
	if formatBug1.Match(data) || formatBug2.Match(data) {
		return 0
	}
	// https://github.com/golang/go/issues/11274
	data = bytes.Replace(data, []byte{'\r'}, []byte{' '}, -1)
	data1, err := format.Source(data)
	if err != nil {
		panic(err)
	}
	if false {
		err = gotypes(data1)
		if err != nil {
			fmt.Printf("new: %q\n", data1)
			fmt.Printf("err: %v\n", err)
			panic("program become invalid after gofmt")
		}
	}
	return 1
}

func gotypes(data []byte) (err error) {
	fset := token.NewFileSet()
	var f *ast.File
	f, err = parser.ParseFile(fset, "src.go", data, parser.ParseComments|parser.DeclarationErrors|parser.AllErrors)
	if err != nil {
		return
	}
	// provide error handler
	// initialize maps in config
	conf := &types.Config{
		Error:    func(err error) {},
		Sizes:    &types.StdSizes{8, 8},
		Importer: importer.For("gc", nil),
	}
	_, err = conf.Check("pkg", fset, []*ast.File{f}, nil)
	if err != nil {
		return
	}
	prog := ssa.NewProgram(fset, ssa.BuildSerially|ssa.SanityCheckFunctions|ssa.GlobalDebug)
	prog.Build()
	for _, pkg := range prog.AllPackages() {
		_, err := pkg.WriteTo(ioutil.Discard)
		if err != nil {
			panic(err)
		}
	}
	return
}

func gc(data []byte) error {
	f, err := ioutil.TempFile("", "fuzz.gc")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	f.Close()
	out, err := exec.Command("compile", f.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s\n%s", out, err)
	}
	return nil
}

func gccgo(data []byte) error {
	cmd := exec.Command("gccgo", "-c", "-x", "go", "-O3", "-o", "/dev/null", "-")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s\n%s", out, err)
	}
	return nil
}
