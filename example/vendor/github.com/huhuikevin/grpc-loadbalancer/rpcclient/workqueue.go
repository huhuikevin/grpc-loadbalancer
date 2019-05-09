package rpcclient

import (
	"errors"
	"reflect"
	"sync"
	"time"
)

const (
	maxTimeWaitForWorkQueue = 10 * time.Second
)

var (
	//ErrWorkQueueFull 工作队列满
	ErrWorkQueueFull = errors.New("Workqueue is full")
	//ErrWorkQueueStoped 工作队列已经停止
	ErrWorkQueueStoped = errors.New("Workqueue is stoped")
	//ErrTooManyPendingWorks 等待的worker太多
	ErrTooManyPendingWorks = errors.New("Too many pending works")
	//ErrWaitTimeOut 工作队列已经停止
	ErrWaitTimeOut = errors.New("Wait for an idle worker timeout")
)

type task struct {
	f      reflect.Value
	args   []reflect.Value
	result chan []reflect.Value
}
type worker struct {
	taskQueue chan *task
	quit      chan int
}

type workQueue struct {
	queueSize    int32
	workingSize  int32
	pendingTasks int32
	idleWorker   chan *worker
	mu           sync.Mutex
	stoped       bool
}

func newWorkQueue(size int32) *workQueue {
	queue := &workQueue{
		queueSize:  size,
		idleWorker: make(chan *worker, size),
		mu:         sync.Mutex{},
	}
	return queue
}

//getIdleWorker 获取一个空闲的工作者，如果idleWorker中没有，而且目前的工作者数量小于queueSize
//创建一个新的，否则跳到等待他人释放工作者，如果等待的人数太多，返回错误
func (wq *workQueue) getIdleWorker() (*worker, error) {
	select {
	case w := <-wq.idleWorker:
		if w == nil {
			return nil, ErrWorkQueueStoped
		}
		return w, nil
	default:
		wq.mu.Lock()
		if wq.stoped {
			wq.mu.Unlock()
			return nil, ErrWorkQueueStoped
		}
		if wq.workingSize >= wq.queueSize {
			goto wait
		}
		wq.workingSize++
		wq.mu.Unlock()
		w := &worker{
			taskQueue: make(chan *task),
			quit:      make(chan int),
		}

		w.start()
		return w, nil
	}
wait:
	if wq.pendingTasks >= wq.queueSize {
		wq.mu.Unlock()
		return nil, ErrTooManyPendingWorks
	}
	wq.pendingTasks++
	wq.mu.Unlock()
	w := <-wq.idleWorker
	wq.mu.Lock()
	wq.pendingTasks--
	wq.mu.Unlock()
	return w, nil
}

//stop 从idleWorker中获取所有的工作者，发送quit信息
func (wq *workQueue) stop() {
	wq.mu.Lock()
	if wq.stoped {
		wq.mu.Unlock()
		return
	}
	wq.stoped = true
	wq.mu.Unlock()
	//下面通过idleWorker保证了worker不可能在其他地方使用，不需要加锁
	for i := int32(0); i < wq.workingSize; i++ {
		worker := <-wq.idleWorker
		worker.quit <- 1
		close(worker.quit)
		close(worker.taskQueue)
	}
	close(wq.idleWorker)
}

func (wq *workQueue) putIdleWorker(worker *worker) {
	wq.idleWorker <- worker
}

func (wq *workQueue) executeTask(f reflect.Value, args []reflect.Value) []reflect.Value {
	worker, err := wq.getIdleWorker()
	if err != nil {
		return []reflect.Value{reflect.Value{}, reflect.ValueOf(err)}
	}
	defer wq.putIdleWorker(worker)
	task := &task{
		f:      f,
		args:   args,
		result: make(chan []reflect.Value),
	}
	return worker.execute(task)
}

func (w *worker) execute(t *task) []reflect.Value {
	w.taskQueue <- t
	v := <-t.result
	return v
}

func (w *worker) start() {
	go func() {
		for {
			select {
			case task := <-w.taskQueue:
				if task == nil {
					return
				}
				v := task.f.Call(task.args)
				task.result <- v
			case <-w.quit:
				return
			}
		}
	}()
}
