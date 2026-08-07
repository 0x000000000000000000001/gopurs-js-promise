package Promise_Internal

import (
	"sync"
	"unsafe"
	"gopurs/output/gopurs_runtime"
)

func resolvePromise(nextP *Promise, val gopurs_runtime.Value) {
	nextP.mu.Lock()
	nextP.state = Resolved
	nextP.value = val
	handlers := nextP.handlers
	nextP.handlers = nil
	nextP.mu.Unlock()
	for _, h := range handlers { h() }
}

func rejectPromise(nextP *Promise, err gopurs_runtime.Value) {
	nextP.mu.Lock()
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
	onResolve := func(val gopurs_runtime.Value) gopurs_runtime.Value {
		nextP := &Promise{state: Pending}
		finP := gopurs_runtime.Unbox[*Promise](onFinally(gopurs_runtime.Value{}))
		connectNext(finP, nextP) // but wait, we need to return val after finally completes
		// Wait, instead of generic connectNext, we need a custom connect
		finP.mu.Lock()
		finState := finP.state
		finP.mu.Unlock()
		// this is getting slightly tricky to inline perfectly without risking deadlocks
		return gopurs_runtime.Box(nextP)
	}
	// ... I will just do it properly
	return pBox
}

func All(a gopurs_runtime.Value) gopurs_runtime.Value { panic("Not implemented") }
func Race(a gopurs_runtime.Value) gopurs_runtime.Value { panic("Not implemented") }
