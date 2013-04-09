package goblin

type CodeGen interface {
    EmitReadStruct(data interface{})
}
