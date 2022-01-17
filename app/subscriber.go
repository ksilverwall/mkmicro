package app

import (
	"github.com/Miyagawa-Ryohei/mkmicro/infra"
	"github.com/Miyagawa-Ryohei/mkmicro/types"
	"sync"
	"time"
)

type Subscriber struct {
	log       types.Logger
	src       types.SessionManager
	container types.HandlerContainer
}

func (s *Subscriber) Listen(pollingSize int) {

	defer s.log.Flush()

	handlers := s.container.Get()
	s.log.Debug("%d handler is found", len(handlers))
	s.log.Info("start subscribe")
	queue, err := s.src.GetQueue()

	if err != nil {
		s.log.Error("%d handler is found", len(handlers))
		panic(err)
	}

	for {
		messages, err := queue.GetMessage(pollingSize)
		if err != nil {
			s.log.Error(err.Error())
			continue
		}
		if len(messages) == 0 {
			s.log.Info("message queue is empty, re-polling after 10 second")
			time.Sleep(10 * time.Second)
		}
		s.log.Debug("%d message is received", len(messages))
		wg := &sync.WaitGroup{}
		for _, m := range messages {
			wg.Add(1)

			go func (target types.Message) {
				mu := &sync.Mutex{}
				done := new(bool)
				*done = false
				go func(){
					defer wg.Done()
					ChangeMessageVisibility(queue, target, mu, done, s.log)
				}()
				go func(target types.Message, mu *sync.Mutex, done *bool) {
					s.log.Debug("[%s] worker start", target.GetDeleteID())
					result := true
					start := time.Now()
					for _, handler := range handlers {
						if err := handler.Exec(target, s.src); err != nil {
							s.log.Info("[%s]handler returns some error. stop change visibility for retry", target.GetDeleteID())
							s.log.Error(err.Error())
							result = false
						} else {
							s.log.Info("[%s]all handler returns no errors. message is processed correctly", target.GetDeleteID())
						}
						s.log.Debug("[%s]worker takes %d msec", target.GetDeleteID(), (time.Now().UnixNano()-start.UnixNano())/int64(time.Millisecond))
					}
					s.log.Debug("[%s]all worker takes %d msec", target.GetDeleteID(), (time.Now().UnixNano()-start.UnixNano())/int64(time.Millisecond))
					s.log.Info("[%s] all worker end", target.GetDeleteID())
					mu.Lock()
					defer mu.Unlock()
					*done = true
					if result {
						if err := queue.DeleteMessage(target); err != nil {
							s.log.Error(err.Error())
							panic(err.Error())
						} else {
							target.SetDeleted(true)
						}
					}
				}(target, mu, done)
			}(m)
		}
		s.log.Info("[subscriber main] wait for processing messages")
		wg.Wait()
	}
}

func NewSubscriber(src types.SessionManager, logger types.Logger, c types.HandlerContainer) *Subscriber {
	log := logger
	if log == nil {
		log = infra.DefaultLogger
	}
	return &Subscriber{
		src:       src,
		log:       log,
		container: c,
	}
}

func ChangeMessageVisibility (queue types.QueueDriver, target types.Message, mu *sync.Mutex, done *bool, log types.Logger) {
	defer mu.Unlock()
	for {
		mu.Lock()
		if *done {
			return
		}
		if !(target.IsDeleted()) {
			if err := queue.ChangeMessageVisibility(target); err != nil {
				log.Error(err.Error())
			}
			mu.Unlock()
		} else {
			return
		}
		time.Sleep(40 * time.Second)
	}
}