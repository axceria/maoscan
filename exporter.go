package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

type exporter struct {
	f         *os.File
	out       string
	proxyType  int
	mu        sync.Mutex
}

func (e *exporter) create() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var filename string
	switch e.proxyType {
	case 1:
		filename = "http.txt"
	case 2:
		filename = "socks4.txt"
	case 3:
		filename = "socks5.txt"
	default:
		return fmt.Errorf("invalid proxy type: %d", strconv.Itoa(e.proxyType))
	}

	var err error
	e.f, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (e *exporter) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.f != nil {
		_, _ = e.f.WriteString("\n")
		_ = e.f.Close()
	}
}

func (e *exporter) Add(s string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.f == nil {
		log.Println("File not initialized")
		return
	}

	_, err := e.f.WriteString(s + "\n")
	if err != nil {
		log.Println(err)
	}
}
