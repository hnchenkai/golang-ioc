package ioc

import (
	"fmt"
	"reflect"
	"strings"
)

func toString(v string) *string {
	return &v
}

func toPtr[T any](v T) *T {
	return &v
}

type StrList []string

func (s *StrList) contains(e string) bool {
	for _, a := range *s {
		if a == e {
			return true
		}
	}
	return false
}

func (s *StrList) Append(e string) {
	*s = append(*s, e)
}

func newStrList(a reflect.Type) StrList {
	baseMethods := StrList{}
	isPtr := a.Kind() == reflect.Ptr
	for i := 0; i < a.NumMethod(); i++ {
		method := a.Method(i)
		argList := []string{}
		if isPtr {
			for j := 1; j < method.Type.NumIn(); j++ {
				argList = append(argList, method.Type.In(j).String())
			}
		} else {
			for j := 0; j < method.Type.NumIn(); j++ {
				argList = append(argList, method.Type.In(j).String())
			}
		}
		outList := []string{}

		for j := 0; j < method.Type.NumOut(); j++ {
			outList = append(outList, method.Type.Out(j).String())
		}
		baseMethods.Append(fmt.Sprintf("%s(%s)(%s)", method.Name, strings.Join(argList, ","), strings.Join(outList, ",")))
	}
	return baseMethods
}

func printMethods(a reflect.Type) string {
	return strings.Join(newStrList(a), ",")
}

var base_type = reflect.TypeOf((*beanComponent)(nil)).Elem()
var base_methods = printMethods(base_type)

// 判断a的方法是否都在b的里面了
func isContains(a reflect.Type, b reflect.Type) {
	// 必须要满足基本要求
	if !b.Implements(base_type) {
		panic(fmt.Sprintf("Component struct[*%s] is not extends ioc.Component Methods[%s]", b.Elem().Name(), base_methods))
	}

	// 如果a是接口类型,只要检查一下b是否满足了a的接口需求
	if b.Implements(a) {
		return
	}

	aMethods := newStrList(a)
	bMethods := newStrList(b)

	for _, aMethod := range aMethods {
		if !bMethods.contains(aMethod) {
			panic(fmt.Sprintf("Component struct[*%s] is not implements interface[%s] method [%s]", b.Elem().Name(), a.Name(), aMethod))
		}
	}

}
