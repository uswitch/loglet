package transformers

import (
	"github.com/uswitch/loglet/types"
)

type DefaultFields struct {
	Fields map[string]string
}

func NewDefaultFields(fields map[string]string) *DefaultFields {
	return &DefaultFields{
		Fields: fields,
	}
}

func (p *DefaultFields) Transform(m *types.LogMessage) {
	for k, v := range p.Fields {
		_, ok := m.Fields[k]
		if !ok {
			m.Fields[k] = v
		}
	}
}
