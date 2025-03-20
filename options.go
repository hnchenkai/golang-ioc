package ioc

import "reflect"

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

func (opt *GetOptions) Fill(typeM reflect.Type) {
	if opt.TypeName == nil {
		opt.TypeName = toString(typeM.Name())
	}
	if opt.PkgName == nil {
		opt.PkgName = toString(typeM.PkgPath())
	}
	if opt.BeanName == nil {
		opt.BeanName = toString(*opt.PkgName + ":" + *opt.TypeName)
	}
	if opt.RealTypeName != nil {
		opt.BeanName = toString(*opt.PkgName + ":" + *opt.TypeName + ":" + *opt.RealTypeName)
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
