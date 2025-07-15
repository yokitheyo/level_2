package main

import "fmt"

func main() {
	a := [5]int{76, 77, 78, 79, 80}
	var b []int = a[1:4]
	fmt.Println(b)
	// создаем слайс массива a от 1( включительно ) до 4( НЕ включительно )  элемента
}
