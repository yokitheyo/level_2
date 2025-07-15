/*
программа выведет числа от 1 до 8 в случайном порядке

select ждет, пока один из каналов вернет значение
как только значение появляется в каком-то канале, оно отправляется в канал c
и select ждет следующего значения

если сразу готово несколько каналов, select выбирает один из них случайным образом

будет вызвана функция func chanrecv(c *hchan, ep unsafe.Pointer, block FALSE) которая реализует чтение из канала
параметр block будет равен false,потому что мы не хотим блокировать горутину ( go park ), если канал пустой
* сразу проверяем все кейсы готовы они или нет  *

если оба канала не готовы select горутина блокируется и ждет

при закрытии канала a или b chanrecv возвращает 0 и ok = false, и канал устанавливается в nil
чтобы исключить канал из дальнейших select'ов
*/

package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

func asChan(vs ...int) <-chan int {
	c := make(chan int)
	go func() {
		for _, v := range vs {
			c <- v
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		}
		close(c)
	}()
	runtime.Gosched()
	return c
}

func merge(a, b <-chan int) <-chan int {
	c := make(chan int)
	go func() {
		for {
			select {
			case v, ok := <-a:
				if ok {
					c <- v
				} else {
					a = nil
				}
			case v, ok := <-b:
				if ok {
					c <- v
				} else {
					b = nil
				}
			}
			if a == nil && b == nil {
				close(c)
				return
			}
		}
	}()
	return c
}

func main() {
	rand.Seed(time.Now().Unix())
	a := asChan(1, 3, 5, 7)
	b := asChan(2, 4, 6, 8)
	c := merge(a, b)
	for v := range c {
		fmt.Print(v)
	}
}
