package flags

import (
	"fmt"
	"reflect"
	"strings"
)

type tagProps struct {
	name string
	args map[string]string
}

func Encode(v any) []string {
	args := []string{}
	return apnd(args, reflect.ValueOf(v))
}

func Append(args []string, v any) []string {
	return apnd(args, reflect.ValueOf(v))
}

func apnd(args []string, v reflect.Value) []string {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return args
		}
		return apnd(args, v.Elem())
	}
	if v.Kind() != reflect.Struct {
		return args
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		tag := parseTag(sf.Tag.Get("flag"))
		if tag == nil {
			continue
		}
		args = appendTag(args, tag, v.Field(i))
	}

	return args
}

func appendTag(args []string, tag *tagProps, v reflect.Value) []string {
	if v.IsZero() {
		return args
	}

	if v.Kind() == reflect.Ptr {
		return appendTag(args, tag, v.Elem())
	}

	switch v.Kind() {
	case reflect.Bool:
		return append(args, tag.name)
	case reflect.Slice:
		if v.Len() == 0 {
			return args
		}
		if joiner, ok := tag.args["join"]; ok {
			var b strings.Builder
			b.WriteString(fmt.Sprint(v.Index(0).Interface()))
			for i := 1; i < v.Len(); i++ {
				b.WriteString(joiner)
				b.WriteString(fmt.Sprint(v.Index(i).Interface()))
			}

			return append(args, tag.name, b.String())
		}
		for i := 0; i < v.Len(); i++ {
			args = append(args, tag.name, fmt.Sprint(v.Index(i).Interface()))
		}
		return args
	default:
		return append(args, tag.name, fmt.Sprint(v.Interface()))
	}

}

func parseTag(s string) *tagProps {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	args := map[string]string{}

	for _, p := range parts {
		argParts := strings.SplitN(p, ":", 2)
		key := argParts[0]
		val := ""
		if len(argParts) > 1 {
			val = argParts[1]
		}
		args[key] = val
	}

	return &tagProps{
		name: parts[0],
		args: args,
	}

}
