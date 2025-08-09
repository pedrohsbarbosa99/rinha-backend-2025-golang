package main

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

var bufPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 10_000))
	},
}

func main() {
	const iterations = 1_000_000
	testString := "Teste buffer pool "

	// Teste 1: criar um bytes.Buffer novo toda vez
	start := time.Now()
	for i := 0; i < iterations; i++ {
		buf := bytes.NewBuffer(make([]byte, 0, 10_000))
		buf.WriteString(testString)
		_ = buf.Bytes()
	}
	elapsedNew := time.Since(start)

	// Teste 2: usar sync.Pool para reutilizar bytes.Buffer
	start = time.Now()
	for i := 0; i < iterations; i++ {
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.WriteString(testString)
		_ = buf.Bytes()
		bufPool.Put(buf)
	}
	elapsedPool := time.Since(start)

	fmt.Printf("Tempo criando bytes.Buffer novo: %v\n", elapsedNew)
	fmt.Printf("Tempo usando sync.Pool bytes.Buffer: %v\n", elapsedPool)
}
