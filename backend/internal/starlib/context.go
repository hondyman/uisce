package starlib

import (
	"fmt"
	"time"

	"go.starlark.net/starlark"
)

const dateLayout = "2006-01-02"

// BuildCtx builds a top-level ctx dict from page + named objects, all dynamic.
func BuildCtx(page map[string]interface{}, objects map[string]map[string]interface{}) *starlark.Dict {
	ctx := starlark.NewDict(1)
	_ = ctx.SetKey(starlark.String("page"), MapToDict(page))
	for name, obj := range objects {
		_ = ctx.SetKey(starlark.String(name), MapToDict(obj))
	}
	return ctx
}

// SplitDataIntoPageAndObjects maps a flat/nested input record into:
// - page: scalar fields
// - objects: any top-level map[string]any fields
// If a top-level key "page" exists and is a map, it is merged into page.
func SplitDataIntoPageAndObjects(data map[string]interface{}) (page map[string]interface{}, objects map[string]map[string]interface{}) {
	page = map[string]interface{}{}
	objects = map[string]map[string]interface{}{}
	for k, v := range data {
		if k == "page" {
			if m, ok := v.(map[string]interface{}); ok {
				for pk, pv := range m {
					page[pk] = pv
				}
			}
			continue
		}
		if m, ok := v.(map[string]interface{}); ok {
			objects[k] = m
			continue
		}
		page[k] = v
	}
	return page, objects
}

// MapToDict converts a map[string]interface{} recursively to *starlark.Dict.
func MapToDict(m map[string]interface{}) *starlark.Dict {
	if m == nil {
		return starlark.NewDict(0)
	}
	d := starlark.NewDict(len(m))
	for k, v := range m {
		_ = d.SetKey(starlark.String(k), GoToValue(v))
	}
	return d
}

// GoToValue converts arbitrary Go values to the closest Starlark type.
func GoToValue(v interface{}) starlark.Value {
	switch t := v.(type) {
	case nil:
		return starlark.None
	case starlark.Value:
		return t
	case string:
		return starlark.String(t)
	case bool:
		return starlark.Bool(t)
	case int:
		return starlark.MakeInt(t)
	case int32:
		return starlark.MakeInt(int(t))
	case int64:
		return starlark.MakeInt64(t)
	case float32:
		return starlark.Float(float64(t))
	case float64:
		return starlark.Float(t)
	case time.Time:
		return starlark.String(t.Format(dateLayout))
	case map[string]interface{}:
		return MapToDict(t)
	case []interface{}:
		vals := make([]starlark.Value, 0, len(t))
		for _, item := range t {
			vals = append(vals, GoToValue(item))
		}
		return starlark.NewList(vals)
	default:
		return starlark.String(fmt.Sprint(v))
	}
}
