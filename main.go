package main

import "github.com/vrypan/snapdown/cmd"

var VERSION string

func main() {
	cmd.Version = VERSION
	cmd.Execute()
}
