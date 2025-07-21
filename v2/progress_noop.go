package output

import "context"

// noOpProgress implements Progress with no-op operations (v1 compatible)
type noOpProgress struct {
	total   int
	current int
	color   ProgressColor
	ctx     context.Context
}

// Core progress methods
func (p *noOpProgress) SetTotal(total int)      { p.total = total }
func (p *noOpProgress) SetCurrent(current int)  { p.current = current }
func (p *noOpProgress) Increment(delta int)     { p.current += delta }
func (p *noOpProgress) SetStatus(status string) {}
func (p *noOpProgress) Complete()               {}
func (p *noOpProgress) Fail(err error)          {}
func (p *noOpProgress) Close() error            { return nil }

// v1 compatibility methods for noOpProgress
func (p *noOpProgress) SetColor(color ProgressColor)   { p.color = color }
func (p *noOpProgress) IsActive() bool                 { return false }
func (p *noOpProgress) SetContext(ctx context.Context) { p.ctx = ctx }
