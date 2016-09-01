package types

type LogMessage struct {
	Fields map[string]interface{}
}

type Transformer interface {
	Transform(m *LogMessage)
}
