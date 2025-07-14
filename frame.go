package ws

type Frame struct {
	FIN        bool
	OpCode     byte
	Masked     bool
	PayloadLen uint64
	MaskingKey []byte
	Payload    []byte
}
