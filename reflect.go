package viot
import (
    "reflect"
    "fmt"
)

func typeSelect(v reflect.Value) interface{} {
    switch v.Kind() {
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return v.Int()
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
        return v.Uint()
    case reflect.Float32, reflect.Float64:
        return v.Float()
    case reflect.Bool:
        return v.Bool()
    case reflect.Complex64, reflect.Complex128:
        return v.Complex()
    case reflect.Invalid:
        return nil
    case reflect.String:
        return v.String()
   	case reflect.UnsafePointer:
   		return v.Pointer()
    case reflect.Slice, reflect.Array:
        if v.CanInterface() {
            return v.Interface()
        }
        
        l := v.Len()
        c := v.Cap()
        vet := reflect.SliceOf(v.Elem().Type())
        cv := reflect.MakeSlice(vet, l, c)
        for i:=0; i<l; i++ {
        	cv = reflect.Append(cv, reflect.ValueOf(typeSelect(v.Index(i))))
        }
        return cv.Interface()
    default:
    	//Interface
    	//Map
    	//Struct
    	//Chan
    	//Func
    	//Ptr
        if v.CanInterface() {
            return v.Interface()
        }
    }
    
   panic(fmt.Errorf("viot: 该类型 %s，无法转换为 interface 类型", v.Kind()))
}

func inDirect(v reflect.Value) reflect.Value {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {}
    return v
}

