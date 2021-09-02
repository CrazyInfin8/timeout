# Timeout
[![GoDoc](https://godoc.org/github.com/CrazyInfin8/timeout?status.svg)](https://pkg.go.dev/github.com/CrazyInfin8/timeout?tab=doc) [![GoReportCard](https://goreportcard.com/badge/github.com/crazyinfin8/timeout)](https://goreportcard.com/report/github.com/crazyinfin8/timeout)

Timeout provides readers that can stop what their doing after a certain time.

Intended specifically for `os.Stdin` as reading it can block code until the user enters data

## Installation:

```
go get github.com/crazyinfin8/timeout
```

## Usage

```go
package main

import (
	"fmt"
	"os"
	"time"

    "github.com/crazyinfin8/timeout"
)

var stdin = timeout.NewReader(os.Stdin).WithTimeout(5 * time.Second)

func main() {
	buf := make([]byte, 128)
	fmt.Print("Enter text here: ")
	count, err := stdin.Read(buf)
	if err != nil {
		if err == (timeout.ErrTimeout{}) {
			println("No input received")
		} else {
			println(err.Error())
		}
	} else {
		fmt.Printf("Text entered: %s", string(buf[:count]))
		fmt.Printf("That's %d bytes!\n", count)
	}
}
```
