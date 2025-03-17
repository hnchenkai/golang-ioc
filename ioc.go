package ioc

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"syscall"

	"github.com/sirupsen/logrus"
)

/**
 * 这里写一个说明文档
 * 模块使用 Regist来注册Componet 相当于java的 @Componet
 * bean实例化中 public的Componet成员字段,将会自动装载,可以使用 ioc:"xxx"的方式来自定义资源名字类似@Resource("xxx")
 */

type BeanMode string

const (
	Singleton BeanMode = "singleton"
	Mutil     BeanMode = "muti"
)

var DefaultBeanMode = Singleton

func SetMode(mode BeanMode) {
	DefaultBeanMode = mode
}

type Component struct {
}

// 这个相当于构造函数只能使用一次
func (c *Component) New(...interface{}) error {
	return nil
}

func (c *Component) GracefulStop() {
}

func createComponent(opt *RegistOptions, typeName string) *subComponent {
	p := beanComponentMgr.findComponent(opt.PkgName, &typeName)
	if p == nil {
		return &subComponent{
			typeName: typeName,
			pkgName:  *opt.PkgName,
			opt:      opt,
		}
	}

	// p 没有order opt也没有那么就异常
	if opt.Order == nil && p.opt.Order == nil {
		//都没有那么就提示异常
		panic(fmt.Sprintf("Component typename[%s:%s] is repeat, maybe need Order!=nil", *opt.PkgName, typeName))
	} else if opt.Order == nil {
		// 有人有了那么就不要注册了
		return nil
	} else if p.opt.Order == nil {
		// 新的有,老的没有,那么就替换
		p.opt = opt
	} else if *opt.Order > *p.opt.Order {
		// 新的比老的数字高,那么就不替换
		return nil
	} else if *opt.Order == *p.opt.Order {
		// 新的比老的数字一样,那么就提示异常
		panic(fmt.Sprintf("Component typename[%s:%s] is repeat, Order now is equal", *opt.PkgName, typeName))
	} else {
		// 新的比老的数字低,那么就替换
		p.opt = opt
	}
	return p
}

// 接口类的实现类注册 T 只能是 Interface T2 需要满足[ioc.Component]的要求
func Bind[T any, T2 beanComponent](options ...*RegistOptions) {
	opt := parseOptions(options...)

	// 泛型 T 要求是一个Interface所以这里要找到这个Interface类型
	typeM := reflect.TypeOf((*T)(nil)).Elem()
	if typeM.Kind() != reflect.Interface {
		panic(fmt.Sprintf("Component typename[%s] is [%s] not interface", typeM.Elem().Name(), typeM.Kind()))
	}
	typeName := typeM.Name()
	if opt.PkgName == nil {
		opt.PkgName = toString(typeM.PkgPath())
	}
	unit := createComponent(opt, typeName)
	if unit == nil {
		return
	}

	// 实际数据要找到结构体的指针类型
	bindM := reflect.TypeOf((*T2)(nil)).Elem()
	// 需要检查是否满足 T的接口实现要求
	isContains(typeM, bindM)

	logrus.Infof("Component typename[%s:%s] is Bind", *opt.PkgName, typeName)

	unit.typeClass = bindM.Elem()

	beanComponentMgr.beanComps = append(beanComponentMgr.beanComps, unit)
}

// 注册组件 T 用 *struct 的形式注入 需要满足[ioc.Component]的要求
func Regist[T beanComponent](options ...*RegistOptions) {
	opt := parseOptions(options...)
	typeM := reflect.TypeOf((*T)(nil)).Elem()
	if typeM.Kind() == reflect.Ptr {
		typeM = typeM.Elem()
	}
	typeName := typeM.Name()
	if opt.PkgName == nil {
		opt.PkgName = toString(typeM.PkgPath())
	}
	unit := createComponent(opt, typeName)
	if unit == nil {
		return
	}
	unit.typeClass = typeM

	logrus.Infof("Component typename[%s:%s] is registed", *opt.PkgName, typeName)
	beanComponentMgr.beanComps = append(beanComponentMgr.beanComps, unit)
}

// 按照类型T获取一个bean
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
	tamplateT := reflect.TypeOf((*T)(nil)).Elem()
	opt := parseOptions(options...)
	opt.Fill(tamplateT)
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
