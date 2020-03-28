package util

import (
	"fmt"
	"testing"
)

func TestID(t *testing.T) {
	fmt.Println(ID("hello-world-1234-qwer"))
	fmt.Println(ID("11"))
}
