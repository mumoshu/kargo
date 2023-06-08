package kargo

import (
	"fmt"
	"reflect"
	"strings"
)

var errorf = fmt.Errorf

type KargoValueProvider interface {
	// KargoValue returns the value of the flag
	KargoValue(get GetValue) (string, error)
}

type KargoArgsAppender interface {
	// KargoAppendArgs produces the arguments for the command
	// to plan and apply the configuration.
	KargoAppendArgs(args *Args, key string) (*Args, error)
}

func AppendArgs(args *Args, i any, key string) (*Args, error) {
	if args == nil {
		args = &Args{}
	}

	t, ok := i.(KargoArgsAppender)
	if ok {
		result, err := t.KargoAppendArgs(args, key)
		if err != nil {
			return nil, err
		}

		if result != nil {
			return result, nil
		}
	}

	return appendArgs(args, i, key)
}

func appendArgs(args *Args, i any, tagKey string) (*Args, error) {
	tpeOfStruct := reflect.TypeOf(i)
	valOfStruct := reflect.ValueOf(i)

	if tpeOfStruct.Kind() == reflect.Pointer {
		tpeOfStruct = tpeOfStruct.Elem()
		valOfStruct = reflect.ValueOf(i).Elem()
	}

	if tpeOfStruct.Kind() != reflect.Struct {
		return nil, errorf("expected Struct but got %s", tpeOfStruct.Kind())
	}

	for i := 0; i < tpeOfStruct.NumField(); i++ {
		tpeOfField := tpeOfStruct.Field(i)
		valOfField := valOfStruct.Field(i)

		var err error

		args, err = appendReflectedArgs(args, valOfField, tpeOfField, tagKey)
		if err != nil {
			return nil, err
		}
	}

	return args, nil
}

func appendReflectedArgs(args *Args, value reflect.Value, field reflect.StructField, tagKey string) (*Args, error) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return args, nil
		}

		value = value.Elem()
	}

	t, ok := value.Interface().(KargoArgsAppender)
	if ok {
		return AppendArgs(args, t, tagKey)
	}

	var flag string

	flagAndOpts, ok := field.Tag.Lookup(tagKey)
	if !ok {
		flag = strings.ToLower(field.Name)
	} else if flagAndOpts == "" {
		// In case it's explicitly set to empty,
		// the intent is to not include this field into args at all.
		return args, nil
	}

	if f, ok := field.Tag.Lookup(FieldTagKargo); ok && f == "" {
		// In case it's explicitly set to empty,
		// the intent is to not include this field into args at all.
		return args, nil
	}

	var v interface{}

	if t, ok := value.Interface().(KargoValueProvider); ok {
		v = t
	} else if value.Kind() == reflect.Struct {
		return appendArgs(args, value.Interface(), tagKey)
	} else if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
		return appendArgs(args, value.Elem().Interface(), tagKey)
	} else if value.Kind() == reflect.Slice {
		var err error
		for i := 0; i < value.Len(); i++ {
			v := value.Index(i)
			args, err = appendReflectedArgs(args, v, field, tagKey)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", field.Name, err)
			}
		}
		return args, nil
	} else {
		v = fmt.Sprintf("%v", value.Interface())

		if v == "" {
			return args, nil
		}

		if strings.HasSuffix(field.Name, "From") {
			v = DynArg{FromOutput: v.(string)}
		}
	}

	if flagAndOpts != "" {
		items := strings.Split(flagAndOpts, ",")
		if len(items) == 2 && items[1] == "arg" {
			return args.Append(v), nil
		}

		if len(items) == 2 && items[1] == "paramless" {
			return args.Append(fmt.Sprintf("--%s", items[0])), nil
		}

		flag = items[0]
	}

	return args.Append(fmt.Sprintf("--%s", flag)).Append(v), nil
}
