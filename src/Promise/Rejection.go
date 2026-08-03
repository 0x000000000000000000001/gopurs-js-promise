package Promise_Rejection

import (
	"reflect"
)

func _ToError(just func(interface{}) interface{}, nothing interface{}, ref interface{}) interface{} {
	val := reflect.ValueOf(ref)
	if val.IsValid() && val.Type().Name() == "Value" {
		method := val.MethodByName("AnyVal")
		if method.IsValid() {
			res := method.Call(nil)
			if len(res) > 0 {
				ref = res[0].Interface()
			}
		}
	}

	if err, ok := ref.(error); ok {
		return just(err)
	}
	return nothing
}

func FromError(err interface{}) interface{} {
	return err
}
