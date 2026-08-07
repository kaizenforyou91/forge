package container

import "reflect"

// AutoWire injects registered services into exported struct fields.
func (c *Container) AutoWire(target any) error {

	v := reflect.ValueOf(target)

	if v.Kind() != reflect.Pointer || v.IsNil() {
		return ErrInvalidTarget
	}

	v = v.Elem()

	if v.Kind() != reflect.Struct {
		return ErrInvalidTarget
	}

	for i := 0; i < v.NumField(); i++ {

		field := v.Field(i)

		if !field.CanSet() {
			continue
		}

		t := field.Type()

		provider, ok := c.services[t]

		if !ok {
			continue
		}

		field.Set(reflect.ValueOf(provider.Instance))
	}

	return nil
}
