package processor

import (
	"gorinha/internal/database"
	"sync"
)

func AddToQueue(db *database.MemClient, bodyQueue chan []byte, pool *sync.Pool) {
	for {
		body := <-bodyQueue
		db.Enqueue(body)
		pool.Put(body)
	}

}
