package output

import "context"

// noOpProgress implements Progress with no-op operations (v1 compatible)
// This is a true no-op implementation that doesn't store any state
type noOpProgress struct{}

// Core progress methods - all true no-ops
func (p *noOpProgress) SetTotal(total int)     {}
func (p *noOpProgress) SetCurrent(current int) {}
func (p *noOpProgress) Increment(delta int)    {}
func (p *noOpProgress) SetStatus(_ string)     {}
func (p *noOpProgress) Complete()              {}
func (p *noOpProgress) Fail(_ error)           {}
func (p *noOpProgress) Close() error           { return nil }

// v1 compatibility methods for noOpProgress - all true no-ops
func (p *noOpProgress) SetColor(color ProgressColor)   {}
func (p *noOpProgress) IsActive() bool                 { return false }
func (p *noOpProgress) SetContext(ctx context.Context) {}
