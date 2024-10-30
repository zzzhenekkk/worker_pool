package workerpool

import (
	"fmt"
	"sort"
	"sync"
)

// Реализовать примитивный worker-pool с возможностью динамически добавлять и удалять воркеров.
// Входные данные (строки) поступают в канал, воркеры их обрабатывают
// (например, выводят на экран номер воркера и сами данные).

type WorkerPool struct {
	input       chan string
	workers     map[int]chan struct{}
	workerCount int
	mu          sync.Mutex
	wg          sync.WaitGroup
	stopOnce    sync.Once
}

func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		input:   make(chan string),
		workers: make(map[int]chan struct{}),
	}
}

func (wp *WorkerPool) AddWorker() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.workerCount++
	id := wp.workerCount

	stopChan := make(chan struct{})
	wp.workers[id] = stopChan
	wp.wg.Add(1)

	go wp.startWorker(id, stopChan)
}

func (wp *WorkerPool) startWorker(workerID int, stopChan chan struct{}) {
	defer wp.wg.Done()
	for {
		select {
		case data, ok := <-wp.input:
			if !ok {
				wp.stopAll()
				return
			}
			fmt.Printf("Воркер %d обрабатывает данные: %s\n", workerID, data)
		case <-stopChan:
			fmt.Printf("Воркер %d останавливается\n", workerID)
			return
		}
	}
}

func (wp *WorkerPool) RemoveWorker(id int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if stopChan, ok := wp.workers[id]; ok {
		close(stopChan)
		delete(wp.workers, id)
	}
}

func (wp *WorkerPool) RemoveWorkerFirstActive() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	keys := make([]int, 0, len(wp.workers))
	for k := range wp.workers {
		keys = append(keys, k)
	}

	if len(keys) == 0 {
		return
	}

	sort.Ints(keys)

	id := keys[0]

	if stopChan, ok := wp.workers[id]; ok {
		close(stopChan)
		delete(wp.workers, id)
	}
}

func (wp *WorkerPool) CloseWP() {
	wp.stopAll()
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

func (wp *WorkerPool) CountActiveWorkers() int {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	return len(wp.workers)
}

func (wp *WorkerPool) InputChannel() chan<- string {
	return wp.input
}

func (wp *WorkerPool) stopAll() {
	wp.stopOnce.Do(func() {
		wp.mu.Lock()
		defer wp.mu.Unlock()

		close(wp.input)

		for id, stopChan := range wp.workers {
			close(stopChan)
			fmt.Printf("Останавливаем воркера %d\n", id)
		}

		wp.workers = make(map[int]chan struct{})
		wp.workerCount = 0
	})
}
