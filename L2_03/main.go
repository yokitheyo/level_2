//Что выведет программа?
//
//Объяснить внутреннее устройство интерфейсов и их отличие от пустых интерфейсов.

package main

import (
	"fmt"
	"os"
)

func Foo() error {
	var err *os.PathError = nil
	return err
}

func main() {
	err := Foo()
	fmt.Println(err)        // значение внутри интерфейса nil, но сам интерфейс не nil -> указывает ссылку на тип os.PathError
	fmt.Println(err == nil) // интерфейс не nil( не пустой ), потому что указывает на тип os.PathError
}
