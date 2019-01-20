package unsafe

import (
	"github.com/viant/toolbox"
	"reflect"
	"strings"
	"unsafe"
)

func init() {
	_= toolbox.SetUnexportedFieldHandler(func(structField *toolbox.StructField) bool {
		fieldValue := structField.Value
		isPointer := fieldValue.Kind() == reflect.Ptr
		if strings.Contains(structField.Type.PkgPath, "github.com/viant/") {
			return false
		}
		if ! fieldValue.CanAddr() &&  ! fieldValue.CanInterface() {
			return false
		}
		ptr := unsafe.Pointer(fieldValue.UnsafeAddr())
		fieldValue = reflect.NewAt(fieldValue.Type(), ptr)
		if isPointer {
			newPointerValue := reflect.New(structField.Value.Type()).Elem()
			reflect.Indirect(fieldValue).Set(newPointerValue)
		}
		structField.Value = reflect.ValueOf(fieldValue.Interface())
		return fieldValue.Kind() == reflect.Ptr
	})

	_ = toolbox.SetStructMetaFilter(func(field reflect.StructField) bool {
		if strings.Contains(field.PkgPath, "github.com/viant/") {
			return false
		}
		return !(len(field.Name) == 1 || strings.HasSuffix(field.Name, "_"))
	})
}
