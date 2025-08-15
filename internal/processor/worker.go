package processor

import (
	"gorinha/internal/database"
	"sync"
)

func AddToQueue(
	pendingQueue chan []byte,
	BodyPool *sync.Pool,
	db *database.Client,
) {
	for {
		body := <-pendingQueue
		db.Enqueue(body)
		BodyPool.Put(body)

	}
}
