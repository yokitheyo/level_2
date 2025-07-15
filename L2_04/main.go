//программа выведет числа от 0 до 9, а потом упадет с дедлоком
//потому что мы читаем из канала в мейн горутине, но никто не пишет в него

package main

func main() {
	ch := make(chan int)
	go func() {
		for i := 0; i < 10; i++ {
			ch <- i
		}
		close(ch)
	}()

	for n := range ch {
		println(n)
	}
}
