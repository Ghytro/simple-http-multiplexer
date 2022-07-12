package limiter

import "sync"

type Limiter struct {
	mutex       sync.Mutex
	connCounter int
	maxConns    int
}

func NewLimiter(maxConns int) *Limiter {
	return &Limiter{maxConns: maxConns}
}

func (l *Limiter) Disconnected() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.connCounter--
}

func (l *Limiter) ConnAllowed() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.connCounter >= l.maxConns {
		return false
	}
	l.connCounter++
	return true
}
