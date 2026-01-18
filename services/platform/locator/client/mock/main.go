package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":51820")
	if err != nil {
		log.Fatal(err)
	}
	ln, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	fmt.Println("Listening on port 51820")
	buf := make([]byte, 1024)
	for {
		n, addr, err := ln.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Received %s from %s \n", string(buf[:n]), addr)
	}
}
