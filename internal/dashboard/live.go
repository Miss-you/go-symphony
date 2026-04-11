package dashboard

import "time"

const minimumIdleRerender = time.Second

type RenderGate struct {
	renderInterval  time.Duration
	lastContent     string
	lastRenderedAt  time.Time
	pendingContent  string
	pendingFlushAt  *time.Time
	lastFingerprint string
}

type RenderDecision struct {
	Render  bool
	Pending bool
	Content string
	FlushAt *time.Time
}

func NewRenderGate(renderInterval time.Duration) *RenderGate {
	if renderInterval <= 0 {
		renderInterval = time.Second
	}
	return &RenderGate{renderInterval: renderInterval}
}

func (g *RenderGate) Enqueue(content string, fingerprint string, now time.Time) RenderDecision {
	if g == nil {
		return RenderDecision{Render: true, Content: content}
	}
	if content == g.lastContent && g.pendingContent == "" {
		if fingerprint == g.lastFingerprint && now.Sub(g.lastRenderedAt) >= minimumIdleRerender {
			g.lastRenderedAt = now
			return RenderDecision{Render: true, Content: content}
		}
		return RenderDecision{}
	}
	if g.lastRenderedAt.IsZero() || now.Sub(g.lastRenderedAt) >= g.renderInterval {
		g.lastContent = content
		g.lastRenderedAt = now
		g.pendingContent = ""
		g.pendingFlushAt = nil
		g.lastFingerprint = fingerprint
		return RenderDecision{Render: true, Content: content}
	}
	flushAt := g.lastRenderedAt.Add(g.renderInterval)
	g.pendingContent = content
	g.pendingFlushAt = &flushAt
	g.lastFingerprint = fingerprint
	return RenderDecision{Pending: true, FlushAt: &flushAt}
}

func (g *RenderGate) Flush(now time.Time) (string, bool) {
	if g == nil || g.pendingFlushAt == nil || g.pendingContent == "" || now.Before(*g.pendingFlushAt) {
		return "", false
	}
	content := g.pendingContent
	g.lastContent = content
	g.lastRenderedAt = now
	g.pendingContent = ""
	g.pendingFlushAt = nil
	return content, true
}

func (g *RenderGate) ForceIdleRerender(fingerprint string, now time.Time) bool {
	if g == nil || g.lastRenderedAt.IsZero() {
		return true
	}
	return fingerprint == g.lastFingerprint && now.Sub(g.lastRenderedAt) >= minimumIdleRerender
}
