package ioc

import (
	"reflect"
	"strings"
)

type GetOptions struct {
	parentBean string

	Lazy *bool
	Args []interface{}

	// BeanName
	BeanName *string
	// 包名
	PkgName *string
	// 泛型或者声明类的名字
	TypeName *string

	// 实现类的名字
	RealTypeName *string
}

// 标签格式 "ioc:[beanName],[lazy],[type=xxx],[bean=xxx],[pkg=xxx]"
func NewGetOption(iocTag string) *GetOptions {
	// 这里提取ioc后面的东西
	iocTag = strings.TrimPrefix(iocTag, "ioc:")
	opt := GetOptions{}
	opt.parseTag(iocTag)
	return &opt
}

// 解析 ioc 标签
// 标签格式 "ioc:beanName,lazy,type=,bean=,pkg="
func (opt *GetOptions) parseTag(iocTag string) {
	iocTagList := strings.Split(iocTag, ",")
	for _, v := range iocTagList {
		if strings.HasPrefix(v, "lazy") {
			opt.Lazy = toPtr(true)
		} else if strings.HasPrefix(v, "pkg=") {
			opt.PkgName = toPtr(strings.TrimPrefix(v, "pkg="))
		} else if strings.HasPrefix(v, "bean=") {
			opt.BeanName = toPtr(strings.TrimPrefix(v, "bean="))
		} else if strings.HasPrefix(v, "type=") {
			opt.RealTypeName = toPtr(strings.TrimPrefix(v, "type="))
		} else if len(v) > 0 {
			opt.BeanName = toPtr(v)
		}
	}
}

func nilSet[T any](ptr **T, val T) {
	if ptr == nil {
		return
	}

	if *ptr == nil {
		*ptr = &val
	}
}

func (opt *GetOptions) parse(unitK reflect.StructField) bool {
	opt.parseTag(unitK.Tag.Get("ioc"))
	if unitK.Type.Kind() == reflect.Interface {
		// 表示是一个接口类,需要按照名字去找
		nilSet(&opt.PkgName, unitK.Type.PkgPath())
		nilSet(&opt.TypeName, unitK.Type.Name())
	} else {
		if _, ok := unitK.Type.MethodByName("New"); !ok {
			return false
		}

		nilSet(&opt.PkgName, unitK.Type.Elem().PkgPath())
		nilSet(&opt.TypeName, unitK.Type.Elem().Name())
	}

	if opt.BeanName == nil {
		if DefaultBeanMode == Singleton {
			opt.BeanName = toPtr(*opt.PkgName + ":" + *opt.TypeName)
		} else {
			opt.BeanName = &unitK.Name
		}
	}
	return true
}

func (opt *GetOptions) Fill(typeM reflect.Type) {
	nilSet(&opt.TypeName, typeM.Name())
	nilSet(&opt.PkgName, typeM.PkgPath())

	nilSet(&opt.BeanName, *opt.PkgName+":"+*opt.TypeName)
	if opt.RealTypeName != nil {
		opt.BeanName = toPtr(*opt.PkgName + ":" + *opt.TypeName + ":" + *opt.RealTypeName)
	}
}

func (c GetOptions) IsOptions() bool {
	return true
}

func (c *GetOptions) IsLazy() bool {
	if c.Lazy == nil {
		return false
	}

	return *c.Lazy
}

func (c *GetOptions) WithArgs(args ...interface{}) *GetOptions {
	c.Args = args
	return c
}

func (c *GetOptions) WithBeanName(name string) *GetOptions {
	c.BeanName = &name
	return c
}

func (c *GetOptions) WithTypeName(name string) *GetOptions {
	c.RealTypeName = &name
	return c
}

func (c *GetOptions) WithLazy() *GetOptions {
	b := true
	c.Lazy = &b
	return c
}

type IOptions interface {
	IsOptions() bool
}

// 懒加载 暂时实现机制问题,还不能很好的使用,不要使用
func Option[T IOptions]() *T {
	return new(T)
}

func parseOptions[T any](options ...*T) *T {
	if len(options) == 1 {
		return options[0]
	}

	opt := new(T)
	if options == nil {
		return opt
	}
	to := reflect.ValueOf(opt)
	for _, v := range options {
		tv := reflect.ValueOf(*v)
		kv := reflect.TypeOf(*v)
		for i := 0; i < tv.NumField(); i++ {
			f := tv.Field(i)
			if !f.IsNil() {
				toF := to.Elem().FieldByName(kv.Field(i).Name)
				if toF.CanSet() {
					toF.Set(f)
				}
			}
		}
	}

	return opt
}

type RegistOptions struct {
	// 是否多强制多实例
	Multi *bool
	// 包名
	PkgName *string
	// 优先级 越小 优先级高
	Order *int
}

func (c RegistOptions) IsOptions() bool {
	return true
}

func (c *RegistOptions) isMulti() bool {
	if c.Multi == nil {
		return false
	}

	return *c.Multi
}

func (c *RegistOptions) WithMulti(b bool) *RegistOptions {
	c.Multi = &b
	return c
}

func (c *RegistOptions) WithPkgName(b string) *RegistOptions {
	c.PkgName = &b
	return c
}

func (c *RegistOptions) WithOrder(b int) *RegistOptions {
	c.Order = &b
	return c
}
