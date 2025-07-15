/*выведет error, потому что переменная err интерфейсного типа
функция test() возвращает nil, но это не nil интерфейса, а nil типа *customError( nil значение, но не nil тип )*/

package main

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func test() *customError {
	// ... do something
	return nil
}

func main() {
	var err error
	err = test()
	if err != nil {
		println("error")
		return
	}
	println("ok")
}
