package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
)

const (
	bytesToMegabytes = (1024.0 * 1024)
	bytesToGigabytes = (1024.0 * 1024.0 * 1024.0)
)

func main() {

	memory, err := memory.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
	fmt.Printf("memory total: %d bytes\n", memory.Total)
	fmt.Printf("memory used: %d bytes\n", memory.Used)
	fmt.Printf("memory cached: %d bytes\n", memory.Cached)
	fmt.Printf("memory free: %d bytes\n", memory.Free)
	fmt.Printf("memory available: %d bytes\n", memory.Available)

	before, err := cpu.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
	time.Sleep(1 * time.Second / 10)
	after, err := cpu.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
	total := float64(after.Total - before.Total)
	fmt.Printf("cpu user: %f %%\n", float64(after.User-before.User)/total*100)
	fmt.Printf("cpu system: %f %%\n", float64(after.System-before.System)/total*100)
	fmt.Printf("cpu idle: %f %%\n", float64(after.Idle-before.Idle)/total*100)

	sFree, sTotal := driveStorage("/")
	totalStorage := float64(sTotal) / bytesToGigabytes
	freeStorage := float64(sFree) / bytesToGigabytes
	sPercent := ((totalStorage - freeStorage) / totalStorage) * 100
	fmt.Printf("Free Storage: %f\n", sPercent)
}

func driveStorage(path string) (free, total uint64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, 0
	}
	// Available blocks * size per block = available space in bytes
	free = uint64(stat.Bavail * uint64(stat.Bsize))
	total = uint64(uint64(stat.Blocks) * uint64(stat.Bsize))

	return free, total
}
