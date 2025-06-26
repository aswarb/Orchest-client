package task

import (
	"context"
	"sync"
)

type Buffer interface {
	Write([]byte) error
	Read() ([]byte, error)
	StartConsume(chan<- []byte, chan<- error, context.Context)
}

func MakeExtensibleBuffer(startLength int) *ExtensibleBuffer {
	arr := make([][]byte, startLength)
	mu := sync.Mutex{}
	readReadyCond := sync.Cond{L: &mu}

	buf := ExtensibleBuffer{
		data:          arr,
		mu:            &mu,
		readReadyCond: &readReadyCond,
	}

	return &buf
}

type ExtensibleBuffer struct {
	mu            *sync.Mutex
	readReadyCond *sync.Cond
	data          [][]byte
}

func (b *ExtensibleBuffer) Write(data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data = append(b.data, data)
	b.readReadyCond.Signal()

	return nil
}

func (b *ExtensibleBuffer) Read() ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for len(b.data) == 0 {
		b.readReadyCond.Wait()
	}

	data := b.data[0]
	b.data = b.data[1:]
	return data, nil
}
