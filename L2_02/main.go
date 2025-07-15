package main

import "fmt"

// возвращаем именнованный параметр
func test() (x int) {
	defer func() {
		x++
	}()
	x = 1
	return
}

// возвращаем анонимный параметр
// defer инкрементирует x, но возвращаемое значение уже определено
func anotherTest() int {
	var x int
	defer func() {
		x++
	}()
	x = 1
	return x
}

func main() {
	defer fmt.Println(test())
	defer fmt.Println(anotherTest())
}
