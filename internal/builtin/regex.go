package builtin

import (
	"context"
	"fmt"
	"regexp"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

var regexStruct *language.Struct

func regex() *language.Struct {
	if regexStruct == nil {
		regexStruct = language.NewStruct("regex", []language.StructField{
			{Name: "pattern", Type: n.TString},
		}, nil)

		sp := regexStruct.GetPrototype().(*language.StructPrototype)
		ctx := context.Background()

		sp.Unlock()
		sp.SetObject(ctx, "init", n.Function(n.Describe(
			n.Arg("self", regexStruct.Type()),
			n.Arg("pattern", n.TString),
		).Returns(regexStruct.Type()),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				proto := self.GetPrototype()
				proto.SetObject(ctx, "pattern", a.Name("pattern"))

				re, err := regexp.Compile(a.Name("pattern").String())
				if err != nil {
					return nil, err
				}

				bucketable, ok := self.(language.Bucketable)
				if ok {
					bucketable.BucketSet("regex", re)
				}

				return self, nil
			}))

		// match
		sp.SetObject(ctx, "match", n.Function(n.Describe(
			n.Arg("self", regexStruct.Type()),
			n.Arg("text", n.TString),
		).Returns(n.TBool),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				text := a.Name("text").String()

				bucketable, _ := self.(language.Bucketable)
				re, _ := bucketable.BucketGet("regex")

				return re.(*regexp.Regexp).MatchString(text), nil
			}))

		// find
		sp.SetObject(ctx, "find", n.Function(n.Describe(
			n.Arg("self", regexStruct.Type()),
			n.Arg("text", n.TString),
		).Returns(n.TString),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				text := a.Name("text").String()

				bucketable, _ := self.(language.Bucketable)
				re, _ := bucketable.BucketGet("regex")

				return re.(*regexp.Regexp).FindString(text), nil
			}))

		// findAll
		sp.SetObject(ctx, "findAll", n.Function(n.Describe(
			n.Arg("self", regexStruct.Type()),
			n.Arg("text", n.TString),
		).Returns(n.TTList(n.TString)),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				text := a.Name("text").String()

				bucketable, ok := self.(language.Bucketable)
				if !ok {
					return nil, fmt.Errorf("object is not bucketable")
				}
				re, _ := bucketable.BucketGet("regex")
				matches := re.(*regexp.Regexp).FindAllString(text, -1)
				var m = make([]any, len(matches))
				for i, match := range matches {
					m[i] = match
				}

				return n.List(m)
			}))

		// replace with optional times
		sp.SetObject(ctx, "replace", n.Function(n.Describe(
			n.Arg("self", regexStruct.Type()),
			n.Arg("text", n.TString),
			n.Arg("replacement", n.TString),
			n.Arg("times", n.TInt, n.Int(-1)), // -1 = replace all
		).Returns(n.TString),
			func(a *n.Args) (any, error) {
				self := a.Name("self")
				text := a.Name("text").String()
				replacement := a.Name("replacement").String()
				times := int(a.Name("times").Value().(int64))

				bucketable, ok := self.(language.Bucketable)
				if !ok {
					return nil, fmt.Errorf("object is not bucketable")
				}
				rawRegex, _ := bucketable.BucketGet("regex")
				re := rawRegex.(*regexp.Regexp)

				if times < 0 {
					return re.ReplaceAllString(text, replacement), nil
				}

				// replace exactly `times` matches
				result := ""
				count := 0
				last := 0
				matches := re.FindAllStringIndex(text, -1)
				for _, m := range matches {
					if count >= times {
						break
					}
					result += text[last:m[0]] + replacement
					last = m[1]
					count++
				}
				result += text[last:] // append the rest
				return result, nil
			}))

		sp.Lock()
		sp.Implement()
	}

	return regexStruct
}
