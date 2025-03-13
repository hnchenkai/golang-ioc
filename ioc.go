package ioc

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

/**
 * 这里写一个说明文档
 * 模块使用 Regist来注册Componet 相当于java的 @Componet
 * bean实例化中 public的Componet成员字段,将会自动装载,可以使用 ioc:"xxx"的方式来自定义资源名字类似@Resource("xxx")
 */

type _BeanComponentMgr struct {
	beanComps []*subComponent
	beanInsts sync.Map
}

var beanComponentMgr _BeanComponentMgr = _BeanComponentMgr{
	beanComps: make([]*subComponent, 0),
	beanInsts: sync.Map{},
}

// 这里假装有一个ioc的控制器,实际是假的,需要手动注册的

type subComponent struct {
	typeName  string
	pkgName   string
	typeClass reflect.Type
	opt       *RegistOptions
}

type BeanMode string

const (
	Singleton BeanMode = "singleton"
	Mutil     BeanMode = "muti"
)

var DefaultBeanMode = Singleton

func SetMode(mode BeanMode) {
	DefaultBeanMode = mode
}

func (c *_BeanComponentMgr) stroeBean(beanName string, bean interface{}, opt *GetOptions) bool {
	newUnit := beanInstance{}
	newUnit.beanName = beanName
	newUnit.bean = bean
	if opt.IsLazy() {
		go func() {
			// 等200毫秒后初始化
			time.Sleep(200 * time.Millisecond)
			newUnit.callInit(opt, c)
		}()
	} else {
		newUnit.callInit(opt, c)
	}
	c.beanInsts.Store(beanName, &newUnit)
	return true
}

func (c *_BeanComponentMgr) loadBean(beanName string) interface{} {
	v, ok := c.beanInsts.Load(beanName)
	if ok {
		return v.(*beanInstance).bean
	} else {
		return nil
	}
}

func (c *_BeanComponentMgr) findComponent(pkgName *string, typeName *string) *subComponent {
	for _, bean := range c.beanComps {
		if pkgName != nil && bean.pkgName != *pkgName {
			continue
		}
		if typeName != nil && bean.typeName != *typeName {
			continue
		}
		return bean
	}
	return nil
}

func (c *_BeanComponentMgr) toNewBean(opt *GetOptions) interface{} {
	bean := c.findComponent(opt.PkgName, opt.TypeName)
	if bean == nil {
		return nil
	}
	return bean.toNew(opt, &beanComponentMgr)
}

type beanInstance struct {
	beanName string
	bean     interface{}
}

type Component struct {
	// inited bool
}

// 这个相当于构造函数只能使用一次
func (c *Component) New(...interface{}) error {
	// if c.inited {
	// 	return errors.New("ioc error, component is inited")
	// }
	// c.inited = true
	return nil
}

func (c *Component) GracefulStop() {
	// do nothing
}

type beanComponent interface {
	New(...interface{}) error
	GracefulStop()
}

// 调用初始化方法
func (bean *beanInstance) callInit(opt *GetOptions, mgr *_BeanComponentMgr) {
	value := bean.bean.(beanComponent)
	// 获取 Component的类型
	vt := reflect.ValueOf(bean.bean).Elem()
	kt := reflect.TypeOf(bean.bean).Elem()
	for i := 0; i < kt.NumField(); i++ {
		unitV := vt.Field(i)
		unitK := kt.Field(i)
		// 不是合格的组件的Pass 组建都是指针类型的
		if _, ok := unitK.Type.MethodByName("New"); !ok {
			continue
		}
		// 不能反射的pass
		if !unitV.CanSet() {
			logrus.Fatalf("ioc error, [%s] field [%s] is not reflectable", kt.Name(), unitK.Name)
		}

		opt := &GetOptions{
			parentBean: bean.beanName,
		}

		opt.PkgName = toString(unitK.Type.Elem().PkgPath())
		opt.TypeName = toString(unitK.Type.Elem().Name())
		if beanName := unitK.Tag.Get("ioc"); len(beanName) != 0 {
			opt.BeanName = &beanName
		} else {
			if DefaultBeanMode == Singleton {
				opt.BeanName = toString(*opt.PkgName + ":" + *opt.TypeName)
			} else {
				opt.BeanName = &unitK.Name
			}
		}
		nB := mgr.loadBean(*opt.BeanName)
		if nB == nil {
			nB = mgr.toNewBean(opt)
		}

		if nB != nil {
			unitV.Set(reflect.ValueOf(nB))
		}
	}

	if err := value.New(opt.Args...); err != nil {
		panic(fmt.Sprintf("bean name [%s] init error %s", bean.beanName, err.Error()))
	} else {
		if len(opt.parentBean) > 0 {
			logrus.WithField("parent", opt.parentBean).Infof("bean name [%s] init success", bean.beanName)
		} else {
			logrus.WithField("parent", "main").Infof("bean name [%s] init success", bean.beanName)
		}
	}
}

// Get 获取bean 没实力化的时候新建初始化
func (bean *subComponent) toNew(opt *GetOptions, mgr *_BeanComponentMgr) interface{} {
	newBean := reflect.New(bean.typeClass).Interface().(beanComponent)
	beanName := ""
	if opt.BeanName == nil {
		if bean.opt.isMulti() {
			return newBean
		}
		beanName = bean.typeName
	} else {
		beanName = *opt.BeanName
	}

	mgr.stroeBean(beanName, newBean, opt)
	return newBean
}

type IOptions interface {
	IsOptions() bool
}

type RegistOptions struct {
	Multi   *bool
	PkgName *string
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

// 注册组件 T 用 *struct 的形式注入
func Regist[T beanComponent](options ...*RegistOptions) {
	opt := parseOptions(options...)
	typeM := reflect.TypeOf(*new(T)).Elem()
	typeName := typeM.Name()
	if opt.PkgName == nil {
		opt.PkgName = toString(typeM.PkgPath())
	}
	if p := beanComponentMgr.findComponent(opt.PkgName, &typeName); p != nil {
		panic(fmt.Sprintf("Component typename[%s:%s] is repeat", *opt.PkgName, typeName))
	}

	logrus.Infof("Component typename[%s:%s] is registed", *opt.PkgName, typeName)
	unit := &subComponent{
		typeName:  typeName,
		pkgName:   *opt.PkgName,
		typeClass: typeM,
		opt:       opt,
	}
	beanComponentMgr.beanComps = append(beanComponentMgr.beanComps, unit)
}

func GetInterface[T any](options ...*GetOptions) T {
	opt := parseOptions(options...)
	if opt.BeanName != nil {
		if bean := beanComponentMgr.loadBean(*opt.BeanName); bean != nil {
			return bean.(T)
		}
	}

	opt.Fill(reflect.TypeOf(*new(T)))
	if result := beanComponentMgr.toNewBean(opt); result != nil {
		return result.(T)
	}

	panic(fmt.Sprintf("component typeName [%s] not found", *opt.TypeName))
}

// 按照类型T获取一个bean 返回的是原始类型 *T
func Get[T any](options ...*GetOptions) (beanOut *T) {
	opt := parseOptions(options...)
	opt.Fill(reflect.TypeOf(*new(T)))
	if bean := beanComponentMgr.loadBean(*opt.BeanName); bean != nil {
		return bean.(*T)
	}

	if result := beanComponentMgr.toNewBean(opt); result != nil {
		return result.(*T)
	}

	panic(fmt.Sprintf("component typeName [%s] not found", *opt.TypeName))
}

// 这里优雅的关闭所有模块
func GracefulStop() {
	beanComponentMgr.beanInsts.Range(func(key, value any) bool {
		// 优雅的退出
		value.(*beanInstance).bean.(beanComponent).GracefulStop()
		return true
	})

	beanComponentMgr.beanInsts.Clear()
}

type GetOptions struct {
	parentBean string

	Lazy *bool
	Args []interface{}

	BeanName *string
	PkgName  *string
	TypeName *string
}

func toString(v string) *string {
	return &v
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

func (c *GetOptions) WithPkgName(name string) *GetOptions {
	c.PkgName = &name
	return c
}

func (c *GetOptions) WithTypeName(name string) *GetOptions {
	c.TypeName = &name
	return c
}

func (c *GetOptions) WithLazy() *GetOptions {
	b := true
	c.Lazy = &b
	return c
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

// 例子
// ioc.Regist("beanUnit", &BeanUnit{})
// 	result := ioc.GetType(BeanUnit{})
// log.Println(result)

var c = make(chan os.Signal, 1)

func Exit(code int) {
	c <- syscall.SIGINT
}

func Run[T any]() {
	// 先初始化第一个模块,然后一次按需加载
	Get[T]()

	// 通过notify系统调用来监听指定的信号值，在这里我们监听了两个信号值：SIGINT和SIGTERM
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	// 阻塞等待信号值的到来
	s := <-c
	// <-time.NewTicker(10 * time.Second).C
	GracefulStop()
	// 输出收到的信号值
	log.Println()
	log.Println(s)

	// 在这里执行你的清理操作
}

// 关闭所有组件重新走初始化流程
func Restart[T any]() {
	GracefulStop()
	logrus.Infoln("current Goroutine", runtime.NumGoroutine())
	Get[T]()
}
