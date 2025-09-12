package main

import (
	"fmt"
	"testing"
)

func TestProtobufLatestTag(t *testing.T) {

	c := make(chan int, 1)
	go func() {
		for c := range c {
			fmt.Println(c)
		}
	}()
	InstallProtoc(c)
}
