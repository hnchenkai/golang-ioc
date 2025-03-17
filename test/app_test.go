package test_test

import (
	"fmt"
	"testing"

	ioc "github.com/hnchenkai/golang-ioc"
)

type IAppComponent interface {
	Hello(a string) string
}

type AppComponent struct {
	ioc.Component
}

func (p *AppComponent) Hello(a string) string {
	return "mock hello"
}

func (p *AppComponent) New(a ...any) error {
	fmt.Println("AppComponent:New")
	return nil
}

type AppComponentImpl struct {
	ioc.Component
}

func (c *AppComponentImpl) Hello(a string) string {
	return "hello"
}

func (c *AppComponentImpl) Hello2(a string) string {
	return "hello2"
}

func (p *AppComponentImpl) New(a ...any) error {
	fmt.Println("AppComponentImpl:New")
	return nil
}

type App struct {
	ioc.Component
	Com  IAppComponent
	Com2 *AppComponent
	// Com3 AppComponent
}

func TestXxx(t *testing.T) {
	ioc.Bind[IAppComponent, *AppComponentImpl]()
	ioc.Bind[IAppComponent, *AppComponent]((&ioc.RegistOptions{}).WithOrder(1))
	ioc.Regist[*AppComponent]()
	ioc.Regist[*App]()
	// 获取第一个对象试试
	ioc.Get[App]()
}
