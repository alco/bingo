package bingo

type CodeGen interface {
	EmitReadStruct(data interface{})
}
