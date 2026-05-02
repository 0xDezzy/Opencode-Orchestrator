package app

import "time"

type RuntimeEvent struct {
	Type, RunID, IssueID string
	Payload              any
	CreatedAt            time.Time
}
type EventBus interface {
	Publish(RuntimeEvent)
	Subscribe(int) (<-chan RuntimeEvent, func())
}
type Bus struct {
	subs map[chan RuntimeEvent]struct{}
	add  chan chan RuntimeEvent
	del  chan chan RuntimeEvent
	pub  chan RuntimeEvent
}

func NewBus() *Bus {
	b := &Bus{subs: map[chan RuntimeEvent]struct{}{}, add: make(chan chan RuntimeEvent), del: make(chan chan RuntimeEvent), pub: make(chan RuntimeEvent, 256)}
	go b.loop()
	return b
}
func (b *Bus) loop() {
	for {
		select {
		case ch := <-b.add:
			b.subs[ch] = struct{}{}
		case ch := <-b.del:
			delete(b.subs, ch)
			close(ch)
		case ev := <-b.pub:
			for ch := range b.subs {
				select {
				case ch <- ev:
				default:
				}
			}
		}
	}
}
func (b *Bus) Publish(ev RuntimeEvent) {
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now()
	}
	select {
	case b.pub <- ev:
	default:
	}
}
func (b *Bus) Subscribe(n int) (<-chan RuntimeEvent, func()) {
	ch := make(chan RuntimeEvent, n)
	b.add <- ch
	return ch, func() { b.del <- ch }
}
