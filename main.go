package main

import "fmt"

func main() {
	keepAlive := initWebRTC()

	fmt.Println("yo yo we're in main")

	<-keepAlive
}
