package web

import (
	"reflect"
	"strconv"
	"errors"

	"IB1/db"
	"github.com/labstack/echo/v4"
)

var errInvalidArgs = errors.New("invalid arguments")

func parsePrivileges(c echo.Context) []string {
	privileges := []string{}
	for _, v := range db.GetPrivileges() {
		priv, _ := getPostForm(c, v)
		if priv == "on" {
			privileges = append(privileges, v)
		}
	}
	return privileges
}

func parseArgument(c echo.Context, name string, kind reflect.Kind) (reflect.Value, error) {
	if name == "privileges" {
		return reflect.ValueOf(parsePrivileges(c)), nil
	}
	v, _ := getPostForm(c, name)
	if v == "" {
		v = c.Param(name)
	}
	nilValue := reflect.ValueOf(nil)
	switch kind {
	case reflect.Bool:
		b := v == "on"
		return reflect.ValueOf(b), nil
	case reflect.Int, reflect.Uint:
		iVal, err := strconv.Atoi(v)
		if err != nil {
			return nilValue, err
		}
		if kind == reflect.Uint {
			return reflect.ValueOf(uint(iVal)), nil
		}
		return reflect.ValueOf(iVal), nil
	case reflect.String:
		return reflect.ValueOf(v), nil
	case reflect.Slice:
		file, err := c.FormFile("banner")
		if err != nil {
			return nilValue, err
		}
		data := make([]byte, file.Size)
		f, err := file.Open()
		if err != nil {
			return nilValue, err
		}
		defer f.Close()
		_, err = f.Read(data)
		if err != nil {
			return nilValue, err
		}
		return reflect.ValueOf(data), nil
	case reflect.Pointer:
		file, err := c.FormFile(name)
		if err != nil {
			return nilValue, err
		}
		return reflect.ValueOf(file), nil
	}
	return nilValue, errInvalidArgs
}

func generic(f any, args ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		funcValue := reflect.ValueOf(f)
		if funcValue.Kind() != reflect.Func {
			return errInvalidArgs
		}
		if err := c.Request().ParseForm(); err != nil {
			return err
		}
		t := funcValue.Type()
		argv := make([]reflect.Value, t.NumIn())
		if len(argv) != len(args) {
			return errInvalidArgs
		}
		for i := range argv {
			v, err := parseArgument(c, args[i], t.In(i).Kind())
			if err != nil {
				return err
			}
			argv[i] = v
		}
		results := funcValue.Call(argv)
		if len(results) != 1 {
			return errInvalidArgs
		}
		v := results[0].Interface()
		switch v.(type) {
		case error:
			return v.(error)
		default:
			return nil
		}
	}
}
