package clock

import "time"

// AdaptiveTicker fires on a fixed interval but can be reset (election timer).
type AdaptiveTicker struct {
	interval time.Duration
	timer    *time.Timer
	ch       chan time.Time
	stop     chan struct{}
}

func NewAdaptiveTicker(interval time.Duration) *AdaptiveTicker {
	a := &AdaptiveTicker{
		interval: interval,
		timer:    time.NewTimer(interval),
		ch:       make(chan time.Time, 1),
		stop:     make(chan struct{}),
	}
	go a.loop()
	return a
}

func (a *AdaptiveTicker) loop() {
	for {
		select {
		case t := <-a.timer.C:
			select {
			case a.ch <- t:
			default:
			}
			a.timer.Reset(a.interval)
		case <-a.stop:
			if !a.timer.Stop() {
				select {
				case <-a.timer.C:
				default:
				}
			}
			return
		}
	}
}

func (a *AdaptiveTicker) C() <-chan time.Time { return a.ch }

func (a *AdaptiveTicker) Reset(d time.Duration) {
	if d > 0 {
		a.interval = d
	}
	if !a.timer.Stop() {
		select {
		case <-a.timer.C:
		default:
		}
	}
	a.timer.Reset(a.interval)
}

func (a *AdaptiveTicker) Stop() {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
}
