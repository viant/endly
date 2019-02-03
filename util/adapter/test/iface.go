package test

type MyInterface interface {

	Func1(param1 string, param2 int)

	Func2(param1 []byte, param2 int) (error)

	Func3(param1 []byte, param2 int) (interface{}, error)

}
