/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import "github.com/vrypan/snapsnapdown/cmd"

var VERSION string

func main() {
	cmd.Version = VERSION
	cmd.Execute()
}
