package Test_Main

import (
	"errors"
	"time"
	"gopurs/output/Promise.Internal"
	"gopurs/output/gopurs_runtime"
)

var Delay = gopurs_runtime.Func(func(msBox gopurs_runtime.Value) gopurs_runtime.Value {
	ms := gopurs_runtime.Unbox[int](msBox)
	return gopurs_runtime.Func(func(_ gopurs_runtime.Value) gopurs_runtime.Value {
		pBox, res, _ := Promise_Internal.NewPromise()
		go func() {
			time.Sleep(time.Duration(ms) * time.Millisecond)
			res(gopurs_runtime.Value{})
		}()
		return pBox
	})
})

var FailAfter = gopurs_runtime.Func(func(msBox gopurs_runtime.Value) gopurs_runtime.Value {
	ms := gopurs_runtime.Unbox[int](msBox)
	return gopurs_runtime.Func(func(_ gopurs_runtime.Value) gopurs_runtime.Value {
		pBox, _, rej := Promise_Internal.NewPromise()
		go func() {
			time.Sleep(time.Duration(ms) * time.Millisecond)
			rej(gopurs_runtime.Box(errors.New("fail")))
		}()
		return pBox
	})
})
