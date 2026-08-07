package Promise_Internal

import (
	"sync"
	"unsafe"
	"gopurs/output/gopurs_runtime"
)

type PromiseState int

const (
	Pending PromiseState = iota
	Resolved
	Rejected
)

type Promise struct {
	state    PromiseState
	value    gopurs_runtime.Value
	err      gopurs_runtime.Value
	handlers []func()
	mu       sync.Mutex
}

func resolvePromise(nextP *Promise, val gopurs_runtime.Value) {
	nextP.mu.Lock()
	if nextP.state != Pending {
		nextP.mu.Unlock()
		return
	}
	nextP.state = Resolved
	nextP.value = val
	handlers := nextP.handlers
	nextP.handlers = nil
	nextP.mu.Unlock()
	for _, h := range handlers { h() }
}

func rejectPromise(nextP *Promise, err gopurs_runtime.Value) {
	nextP.mu.Lock()
	if nextP.state != Pending {
		nextP.mu.Unlock()
		return
	}
	nextP.state = Rejected
	nextP.err = err
	handlers := nextP.handlers
	nextP.handlers = nil
	nextP.mu.Unlock()
	for _, h := range handlers { h() }
}

func connectNext(next, nextP *Promise) {
	if next == nil { panic("next is nil!") }
	next.mu.Lock()
	if next.state == Resolved {
		next.mu.Unlock()
		resolvePromise(nextP, next.value)
	} else if next.state == Rejected {
		next.mu.Unlock()
		rejectPromise(nextP, next.err)
	} else {
		next.handlers = append(next.handlers, func() {
			var state PromiseState
			var value, err gopurs_runtime.Value
			next.mu.Lock()
			state, value, err = next.state, next.value, next.err
			next.mu.Unlock()
			if state == Resolved {
				resolvePromise(nextP, value)
			} else {
				rejectPromise(nextP, err)
			}
		})
		next.mu.Unlock()
	}
}

func chainPromise(pBox gopurs_runtime.Value, onResolve, onReject func(gopurs_runtime.Value) gopurs_runtime.Value) gopurs_runtime.Value {
	p := gopurs_runtime.Unbox[*Promise](pBox)
	nextP := &Promise{state: Pending}
	handle := func() {
		var nextBox gopurs_runtime.Value
		if p.state == Resolved {
			if onResolve != nil {
				nextBox = onResolve(p.value)
				connectNext(gopurs_runtime.Unbox[*Promise](nextBox), nextP)
			} else {
				resolvePromise(nextP, p.value)
			}
		} else {
			if onReject != nil {
				nextBox = onReject(p.err)
				connectNext(gopurs_runtime.Unbox[*Promise](nextBox), nextP)
			} else {
				rejectPromise(nextP, p.err)
			}
		}
	}
	p.mu.Lock()
	if p.state == Pending {
		p.handlers = append(p.handlers, handle)
		p.mu.Unlock()
	} else {
		p.mu.Unlock()
		handle()
	}
	return gopurs_runtime.Box(nextP)
}

func New(executorBox gopurs_runtime.Value) gopurs_runtime.Value {
	executor := *(*func(gopurs_runtime.Value, gopurs_runtime.Value) gopurs_runtime.Value)(unsafe.Pointer(&executorBox.UnsafePtr))
	p := &Promise{state: Pending}
	resolve := gopurs_runtime.Func(func(val gopurs_runtime.Value) gopurs_runtime.Value {
		resolvePromise(p, val)
		return gopurs_runtime.Value{}
	})
	reject := gopurs_runtime.Func(func(err gopurs_runtime.Value) gopurs_runtime.Value {
		rejectPromise(p, err)
		return gopurs_runtime.Value{}
	})
	executor(resolve, reject)
	return gopurs_runtime.Box(p)
}

func NewPromise() (gopurs_runtime.Value, func(gopurs_runtime.Value), func(gopurs_runtime.Value)) {
	p := &Promise{state: Pending}
	resolve := func(val gopurs_runtime.Value) { resolvePromise(p, val) }
	reject := func(err gopurs_runtime.Value) { rejectPromise(p, err) }
	return gopurs_runtime.Box(p), resolve, reject
}

func Resolve(valBox gopurs_runtime.Value) gopurs_runtime.Value {
	p := &Promise{state: Resolved, value: valBox}
	return gopurs_runtime.Box(p)
}

func Reject(errBox gopurs_runtime.Value) gopurs_runtime.Value {
	p := &Promise{state: Rejected, err: errBox}
	return gopurs_runtime.Box(p)
}

func ThenOrCatch(onResolveBox gopurs_runtime.Value, onRejectBox gopurs_runtime.Value, pBox gopurs_runtime.Value) gopurs_runtime.Value {
	onResolve := *(*func(gopurs_runtime.Value) gopurs_runtime.Value)(unsafe.Pointer(&onResolveBox.UnsafePtr))
	onReject := *(*func(gopurs_runtime.Value) gopurs_runtime.Value)(unsafe.Pointer(&onRejectBox.UnsafePtr))
	return chainPromise(pBox, onResolve, onReject)
}

func Then_(onResolveBox gopurs_runtime.Value, pBox gopurs_runtime.Value) gopurs_runtime.Value {
	onResolve := *(*func(gopurs_runtime.Value) gopurs_runtime.Value)(unsafe.Pointer(&onResolveBox.UnsafePtr))
	return chainPromise(pBox, onResolve, nil)
}

func Catch(onRejectBox gopurs_runtime.Value, pBox gopurs_runtime.Value) gopurs_runtime.Value {
	onReject := *(*func(gopurs_runtime.Value) gopurs_runtime.Value)(unsafe.Pointer(&onRejectBox.UnsafePtr))
	return chainPromise(pBox, nil, onReject)
}

func Finally(onFinallyBox gopurs_runtime.Value, pBox gopurs_runtime.Value) gopurs_runtime.Value {
	onFinally := *(*func(gopurs_runtime.Value) gopurs_runtime.Value)(unsafe.Pointer(&onFinallyBox.UnsafePtr))
	p := gopurs_runtime.Unbox[*Promise](pBox)
	nextP := &Promise{state: Pending}
	handle := func() {
		finBox := onFinally(gopurs_runtime.Value{}) 
		finP := gopurs_runtime.Unbox[*Promise](finBox)
		
		finHandle := func() {
			finP.mu.Lock()
			finState, finErr := finP.state, finP.err
			finP.mu.Unlock()
			
			if finState == Rejected {
				rejectPromise(nextP, finErr)
			} else {
				if p.state == Resolved {
					resolvePromise(nextP, p.value)
				} else {
					rejectPromise(nextP, p.err)
				}
			}
		}
		
		finP.mu.Lock()
		if finP.state == Pending {
			finP.handlers = append(finP.handlers, finHandle)
			finP.mu.Unlock()
		} else {
			finP.mu.Unlock()
			finHandle()
		}
	}
	
	p.mu.Lock()
	if p.state == Pending {
		p.handlers = append(p.handlers, handle)
		p.mu.Unlock()
	} else {
		p.mu.Unlock()
		handle()
	}
	return gopurs_runtime.Box(nextP)
}

func All(arrBox gopurs_runtime.Value) gopurs_runtime.Value {
	arr := gopurs_runtime.Unbox[[]gopurs_runtime.Value](arrBox)
	nextP := &Promise{state: Pending}
	if len(arr) == 0 {
		resolvePromise(nextP, gopurs_runtime.Box([]gopurs_runtime.Value{}))
		return gopurs_runtime.Box(nextP)
	}
	
	results := make([]gopurs_runtime.Value, len(arr))
	var count int
	var mu sync.Mutex
	var done bool
	
	for i, pBox := range arr {
		p := gopurs_runtime.Unbox[*Promise](pBox)
		idx := i
		handle := func() {
			p.mu.Lock()
			state, value, err := p.state, p.value, p.err
			p.mu.Unlock()
			
			if state == Rejected {
				mu.Lock()
				if !done {
					done = true
					mu.Unlock()
					rejectPromise(nextP, err)
				} else {
					mu.Unlock()
				}
			} else {
				mu.Lock()
				if !done {
					results[idx] = value
					count++
					if count == len(arr) {
						done = true
						mu.Unlock()
						resolvePromise(nextP, gopurs_runtime.Box(results))
					} else {
						mu.Unlock()
					}
				} else {
					mu.Unlock()
				}
			}
		}
		
		p.mu.Lock()
		if p.state == Pending {
			p.handlers = append(p.handlers, handle)
			p.mu.Unlock()
		} else {
			p.mu.Unlock()
			handle()
		}
	}
	
	return gopurs_runtime.Box(nextP)
}

func Race(arrBox gopurs_runtime.Value) gopurs_runtime.Value {
	arr := gopurs_runtime.Unbox[[]gopurs_runtime.Value](arrBox)
	nextP := &Promise{state: Pending}
	var mu sync.Mutex
	var done bool
	
	for _, pBox := range arr {
		p := gopurs_runtime.Unbox[*Promise](pBox)
		handle := func() {
			p.mu.Lock()
			state, value, err := p.state, p.value, p.err
			p.mu.Unlock()
			
			mu.Lock()
			if !done {
				done = true
				mu.Unlock()
				if state == Resolved {
					resolvePromise(nextP, value)
				} else {
					rejectPromise(nextP, err)
				}
			} else {
				mu.Unlock()
			}
		}
		
		p.mu.Lock()
		if p.state == Pending {
			p.handlers = append(p.handlers, handle)
			p.mu.Unlock()
		} else {
			p.mu.Unlock()
			handle()
		}
	}
	
	return gopurs_runtime.Box(nextP)
}
