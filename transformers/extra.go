package transformers

import (
	"github.com/uswitch/loglet/types"
)

type ExtraFields struct {
	Fields map[string]string
}

func NewExtraFields(fields map[string]string) *ExtraFields {
	return &ExtraFields{
		Fields: fields,
	}
}

func (p *ExtraFields) Transform(m *types.LogMessage) {
	for k, v := range p.Fields {
		m.Fields[k] = v
	}
}
