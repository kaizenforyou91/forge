package container

import "reflect"

// Constructor describes a constructor function.
type Constructor struct {
	Input []reflect.Type

	Output reflect.Type

	Factory reflect.Value
}

// ParseConstructor parses a constructor function.
func ParseConstructor(fn any) (Constructor, error) {

	t := reflect.TypeOf(fn)

	if t.Kind() != reflect.Func {
		return Constructor{}, ErrInvalidConstructor
	}

	if t.NumOut() != 1 {
		return Constructor{}, ErrInvalidConstructor
	}

	c := Constructor{

		Output: t.Out(0),

		Factory: reflect.ValueOf(fn),
	}

	for i := 0; i < t.NumIn(); i++ {

		c.Input = append(c.Input, t.In(i))

	}

	return c, nil

}

// Call executes the constructor.
func (c Constructor) Call(args []reflect.Value) (any, error) {

	results := c.Factory.Call(args)

	return results[0].Interface(), nil
}
