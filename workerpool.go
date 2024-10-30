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
	dataHandler DataProcessor
}

// DataProcessor определяет тип функции для обработки данных с идентификатором воркера.
type DataProcessor func(workerID int, data interface{})

// defaultDataHandler - дефолтный обработчик, обрабатывает строки, выводя их на экран
func defaultDataHandler(workerID int, data interface{}) {
	if str, ok := data.(string); ok {
		fmt.Printf("Вокрер %d обрабатывает данные: %s\n", workerID, str)
	} else {
		fmt.Printf("Вокрер %d, данные не являются строкой: %v\n", workerID, data)
	}
}

func NewWorkerPool(processor ...DataProcessor) *WorkerPool {
	wp := &WorkerPool{
		input:   make(chan string),
		workers: make(map[int]chan struct{}),
	}

	if len(processor) == 0 {
		wp.dataHandler = defaultDataHandler
	} else {
		wp.dataHandler = processor[0]
	}

	return wp
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
			wp.dataHandler(workerID, data)
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
