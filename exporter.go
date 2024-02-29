package main

import (
	"log"
	"os"
	"sync"
)

type Exporter struct {
	f         *os.File
	out       string
	proxyType  int
	mu        sync.Mutex
}

func (e *Exporter) create() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var err error
	e.f, err = os.OpenFile(e.out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (e *Exporter) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.f != nil {
		_, _ = e.f.WriteString("\n")
		_ = e.f.Close()
	}
}

func (e *Exporter) Add(s string) {
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
