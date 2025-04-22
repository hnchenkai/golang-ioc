package ioc

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

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
	// 	这里对之前注册的都存放一下,主要是给后续指定模式使用
	pool []*subComponent
}

func (sc *subComponent) pushSubComponent(sub *subComponent) {
	sc.pool = append(sc.pool, sub)
}

func (sc *subComponent) findSubComponent(typename string) *subComponent {
	for _, v := range sc.pool {
		if v.typeName == typename {
			return v
		}
	}
	return nil
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
	if opt.RealTypeName != nil {
		// 如果指定了资源名字,那么就需要去找特定的
		if bean = bean.findSubComponent(*opt.RealTypeName); bean == nil {
			return nil
		}
	}

	return bean.toNew(opt, &beanComponentMgr)
}

type beanInstance struct {
	beanName string
	bean     interface{}
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
		opt := &GetOptions{
			parentBean: bean.beanName,
			Args:       opt.Args,
		}

		if !opt.parse(unitK) {
			continue
		}

		// 不能反射的pass
		if !unitV.CanSet() {
			logrus.Fatalf("ioc error, [%s] field [%s] is not reflectable", kt.Name(), unitK.Name)
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
