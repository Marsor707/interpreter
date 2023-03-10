package evaluator

import (
	"fmt"
	"interpreter/object"
)

var builtins = map[string]*object.Builtin{
	"len": {
		func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("入参数量不正确，需要1个，实际%d个", len(args))
			}
			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			default:
				return newError("len不支持的参数类型，%s", args[0].Type())
			}
		},
	},
	"first": {
		func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("入参数量不正确，需要1个，实际%d个", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("first不支持的参数类型，%s", args[0].Type())
			}
			arr := args[0].(*object.Array)
			if len(arr.Elements) > 0 {
				return arr.Elements[0]
			}
			return NULL
		},
	},
	"last": {
		func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("入参数量不正确，需要1个，实际%d个", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("first不支持的参数类型，%s", args[0].Type())
			}
			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				return arr.Elements[length-1]
			}

			return NULL
		},
	},
	"rest": {
		func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("入参数量不正确，需要1个，实际%d个", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("first不支持的参数类型，%s", args[0].Type())
			}
			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				newElements := make([]object.Object, length-1)
				copy(newElements, arr.Elements[1:])
				return &object.Array{Elements: newElements}
			}
			return NULL
		},
	},
	"push": {
		func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("入参数量不正确，需要2个，实际%d个", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("first不支持的参数类型，%s", args[0].Type())
			}
			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			newElements := make([]object.Object, length+1)
			copy(newElements, arr.Elements)
			newElements[length] = args[1]
			return &object.Array{Elements: newElements}
		},
	},
	"puts": {
		func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}
			return NULL
		},
	},
}
