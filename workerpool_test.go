package workerpool

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	wp := NewWorkerPool()

	wp.AddWorker()
	wp.AddWorker()

	go func() {
		for i := 0; i < 5; i++ {
			wp.InputChannel() <- "Тестовые данные"
		}
		wp.CloseWP()
	}()

	wp.Wait()

	if active := wp.CountActiveWorkers(); active != 0 {
		t.Errorf("ожидалось 0 активных воркеров, получено %d", active)
	}
}

func TestWorkerPoolBasic(t *testing.T) {
	wp := NewWorkerPool()

	wp.AddWorker()
	wp.AddWorker()
	wp.AddWorker()

	var wg sync.WaitGroup
	numTasks := 10

	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		wp.InputChannel() <- "Task " + strconv.Itoa(i)
	}

	wp.CloseWP()
	wp.Wait()

	if wp.CountActiveWorkers() != 0 {
		t.Errorf("Expected 0 active workers, got %d", wp.CountActiveWorkers())
	}
}

func TestWorkerAdditionAndRemoval(t *testing.T) {
	wp := NewWorkerPool()

	wp.AddWorker()
	wp.AddWorker()

	if wp.CountActiveWorkers() != 2 {
		t.Errorf("Expected 2 active workers, got %d", wp.CountActiveWorkers())
	}

	wp.RemoveWorker(1)

	if wp.CountActiveWorkers() != 1 {
		t.Errorf("Expected 1 active worker, got %d", wp.CountActiveWorkers())
	}

	wp.AddWorker()

	if wp.CountActiveWorkers() != 2 {
		t.Errorf("Expected 2 active workers, got %d", wp.CountActiveWorkers())
	}

	for i := 0; i < 5; i++ {
		wp.InputChannel() <- "Task " + strconv.Itoa(i)
	}

	time.Sleep(1 * time.Second)

	wp.CloseWP()
	wp.Wait()
}

func TestCloseWhileRunning(t *testing.T) {
	wp := NewWorkerPool()

	wp.AddWorker()
	wp.AddWorker()
	wp.AddWorker()

	for i := 0; i < 100; i++ {
		wp.InputChannel() <- "Task " + strconv.Itoa(i)
		time.Sleep(10 * time.Millisecond)
	}

	wp.CloseWP()

	wp.Wait()

	if wp.CountActiveWorkers() != 0 {
		t.Errorf("Expected 0 active workers, got %d", wp.CountActiveWorkers())
	}
}

func TestMultipleCloseWP(t *testing.T) {
	wp := NewWorkerPool()

	wp.AddWorker()

	// Закрываем пул несколько раз
	wp.CloseWP()
	wp.CloseWP()
	wp.Wait()

	if wp.CountActiveWorkers() != 0 {
		t.Errorf("Expected 0 active workers after multiple CloseWP calls, got %d", wp.CountActiveWorkers())
	}
}

func TestWorkerPoolNoWorkers(t *testing.T) {
	wp := NewWorkerPool()

	// Отправляем задачу без воркеров
	doneCh := make(chan struct{})

	go func() {
		wp.InputChannel() <- "Task without workers"
		close(doneCh)
	}()

	select {
	case <-doneCh:
		t.Error("Expected to block when sending task without workers")
	case <-time.After(100 * time.Millisecond):
		// Ожидаем, что задача блокируется, так как нет воркеров
	}

	// Добавляем воркера
	wp.AddWorker()

	// Теперь задача должна быть обработана
	select {
	case <-doneCh:
		// Задача обработана
	case <-time.After(1 * time.Second):
		t.Error("Task was not processed after adding a worker")
	}

	// Закрываем пул
	wp.CloseWP()
	wp.Wait()
}

func TestRemoveWorkerFirstActive(t *testing.T) {
	wp := NewWorkerPool()

	// Добавляем 3 воркера с ID 1, 2, 3
	wp.AddWorker() // ID 1
	wp.AddWorker() // ID 2
	wp.AddWorker() // ID 3

	if wp.CountActiveWorkers() != 3 {
		t.Errorf("Expected 3 active workers, got %d", wp.CountActiveWorkers())
	}

	// Удаляем первого активного воркера
	wp.RemoveWorkerFirstActive()

	if wp.CountActiveWorkers() != 2 {
		t.Errorf("Expected 2 active workers after removal, got %d", wp.CountActiveWorkers())
	}

	// Проверяем, что воркер с ID 1 был удалён
	wp.mu.Lock()
	if _, exists := wp.workers[1]; exists {
		t.Errorf("Worker with ID 1 should have been removed")
	}
	wp.mu.Unlock()
}

func TestRemoveWorkerFirstActiveAll(t *testing.T) {
	wp := NewWorkerPool()

	// Добавляем 3 воркера
	wp.AddWorker()
	wp.AddWorker()
	wp.AddWorker()

	totalWorkers := 3

	for i := 0; i < totalWorkers; i++ {
		wp.RemoveWorkerFirstActive()
		expectedWorkers := totalWorkers - i - 1
		if wp.CountActiveWorkers() != expectedWorkers {
			t.Errorf("Expected %d active workers after removal, got %d", expectedWorkers, wp.CountActiveWorkers())
		}
	}

	// Проверяем, что все воркеры удалены
	if wp.CountActiveWorkers() != 0 {
		t.Errorf("Expected 0 active workers, got %d", wp.CountActiveWorkers())
	}
}

func TestRemoveWorkerFirstActiveNoWorkers(t *testing.T) {
	wp := NewWorkerPool()

	// Проверяем, что при отсутствии воркеров метод не вызывает панику
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Method panicked when no workers were present: %v", r)
		}
	}()

	wp.RemoveWorkerFirstActive()
	if wp.CountActiveWorkers() != 0 {
		t.Errorf("Expected 0 active workers, got %d", wp.CountActiveWorkers())
	}
}
