package kargo

import (
	"fmt"
	"reflect"
	"strings"
)

var errorf = fmt.Errorf

type flagValueProvider interface {
	// FlagValue returns the value of the flag
	FlagValue(get GetValue) (string, error)
}

type argsAppender interface {
	// AppendArgs produces the arguments for the command
	// to plan and apply the configuration.
	AppendArgs(args []string, get GetValue, key string) (*[]string, error)
}

func AppendArgs(args []string, i any, get GetValue, key string) ([]string, error) {
	t, ok := i.(argsAppender)
	if ok {
		result, err := t.AppendArgs(args, get, key)
		if err != nil {
			return nil, err
		}

		if result != nil {
			return *result, nil
		}
	}

	return appendArgs(args, i, get, key)
}

func appendArgs(args []string, i any, get GetValue, tagKey string) ([]string, error) {
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

		args, err = appendReflectedArgs(args, valOfField, tpeOfField, get, tagKey)
		if err != nil {
			return nil, err
		}
	}

	return args, nil
}

func appendReflectedArgs(args []string, value reflect.Value, field reflect.StructField, get GetValue, tagKey string) ([]string, error) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return args, nil
		}

		value = value.Elem()
	}

	t, ok := value.Interface().(argsAppender)
	if ok {
		return AppendArgs(args, t, get, tagKey)
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

	var v string

	if t, ok := value.Interface().(flagValueProvider); ok {
		var err error

		v, err = t.FlagValue(get)
		if err != nil {
			return nil, err
		}
	} else if value.Kind() == reflect.Struct {
		return appendArgs(args, value.Interface(), get, tagKey)
	} else if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
		return appendArgs(args, value.Elem().Interface(), get, tagKey)
	} else {
		v = fmt.Sprintf("%v", value.Interface())

		if v == "" {
			return args, nil
		}

		if strings.HasSuffix(field.Name, "From") {
			var err error

			v, err = get(v)
			if err != nil {
				return nil, errorf("field %s: unable to get value: %w", field.Name, err)
			}
		}
	}

	if flagAndOpts != "" {
		items := strings.Split(flagAndOpts, ",")
		if len(items) == 2 && items[1] == "arg" {
			return append(args, items[1]), nil
		}

		flag = items[0]
	}

	return append(args, fmt.Sprintf("--%s=%s", flag, v)), nil
}
