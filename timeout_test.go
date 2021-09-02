package timeout

import (
	"fmt"
	"os"
	"testing"
	"time"
)

var stdin = NewReader(os.Stdin).WithTimeout(5 * time.Second)

func Test(t *testing.T) {
	buf := make([]byte, 128)
	fmt.Print("Enter text here: ")
	count, err := stdin.Read(buf)
	if err != nil {
		if err == (ErrTimeout{}) {
			println("No input received")
		} else {
			println(err.Error())
		}
	} else {
		fmt.Printf("Text entered: %s", string(buf[:count]))
		fmt.Printf("That's %d bytes!\n", count)
	}
}
