// Package netcheck implements the DNS, TLS and HTTP inspection tools exposed
// under /v1/tools. Every tool in this package is a pure function of its input
// plus a Resolver: no HTTP handling, no database access, no logging. That keeps
// the checks testable against fixtures and lets the REST handlers, the MCP
// tools and the background monitoring worker all share one implementation.
package netcheck

import (
	"sort"
	"time"
)

// Status is the verdict of a single finding or of a whole report.
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusInfo Status = "info"
)

// Severity ranks how much a finding matters. It drives both the score
// deduction and the order findings are presented in.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// severityWeight is the score deducted when a finding of this severity fails.
// A warn costs half as much as a fail.
var severityWeight = map[Severity]int{
	SeverityCritical: 40,
	SeverityHigh:     25,
	SeverityMedium:   12,
	SeverityLow:      5,
	SeverityInfo:     0,
}

// severityRank orders findings of equal status, worst first.
var severityRank = map[Severity]int{
	SeverityCritical: 5,
	SeverityHigh:     4,
	SeverityMedium:   3,
	SeverityLow:      2,
	SeverityInfo:     1,
}

// statusRank orders findings so failures surface above warnings above notes.
var statusRank = map[Status]int{
	StatusFail: 4,
	StatusWarn: 3,
	StatusInfo: 2,
	StatusPass: 1,
}

// Finding is one atomic observation. The ID is stable across runs so reports
// can be diffed over time and alerts can be deduplicated; Detail and
// Remediation are the user-facing halves ("what is wrong" / "what to do").
type Finding struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      Status   `json:"status"`
	Severity    Severity `json:"severity"`
	Detail      string   `json:"detail"`
	Remediation string   `json:"remediation,omitempty"`
	Evidence    any      `json:"evidence,omitempty"`
}

// Report is the uniform envelope every tool returns. A single frontend
// component renders all of them, and the scoring, diffing and alerting layers
// work against this shape rather than against per-tool structs.
type Report struct {
	Tool      string    `json:"tool"`
	Target    string    `json:"target"`
	Verdict   Status    `json:"verdict"`
	Score     int       `json:"score"`
	Grade     string    `json:"grade"`
	Summary   string    `json:"summary"`
	Findings  []Finding `json:"findings"`
	Raw       any       `json:"raw,omitempty"`
	TimingMS  int64     `json:"timing_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

// Counts returns how many findings carry each status, for summary lines.
func (r *Report) Counts() (pass, warn, fail int) {
	for _, f := range r.Findings {
		switch f.Status {
		case StatusPass:
			pass++
		case StatusWarn:
			warn++
		case StatusFail:
			fail++
		}
	}
	return
}

// Builder accumulates findings for one tool run and computes the score and
// verdict at the end. Tools construct one with NewReport, call the Pass/Warn/
// Fail/Info helpers as they inspect, then call Build.
type Builder struct {
	tool     string
	target   string
	started  time.Time
	findings []Finding
	raw      any
	summary  string
}

// NewReport starts a report and begins the timing clock.
func NewReport(tool, target string) *Builder {
	return &Builder{tool: tool, target: target, started: time.Now()}
}

// Add appends a pre-built finding, for the cases where Evidence is needed.
func (b *Builder) Add(f Finding) *Builder {
	b.findings = append(b.findings, f)
	return b
}

// Pass records a check that succeeded.
func (b *Builder) Pass(id, title, detail string) *Builder {
	return b.Add(Finding{ID: id, Title: title, Status: StatusPass, Severity: SeverityInfo, Detail: detail})
}

// Info records a neutral observation that does not affect the score.
func (b *Builder) Info(id, title, detail string) *Builder {
	return b.Add(Finding{ID: id, Title: title, Status: StatusInfo, Severity: SeverityInfo, Detail: detail})
}

// Warn records a problem worth fixing that is not currently breaking anything.
func (b *Builder) Warn(id, title string, sev Severity, detail, remediation string) *Builder {
	return b.Add(Finding{ID: id, Title: title, Status: StatusWarn, Severity: sev, Detail: detail, Remediation: remediation})
}

// Fail records a problem that is actively broken or dangerous.
func (b *Builder) Fail(id, title string, sev Severity, detail, remediation string) *Builder {
	return b.Add(Finding{ID: id, Title: title, Status: StatusFail, Severity: sev, Detail: detail, Remediation: remediation})
}

// Raw attaches the unprocessed data the checks were derived from, so the UI
// can offer a "show raw records" panel and API users can do their own analysis.
func (b *Builder) Raw(v any) *Builder {
	b.raw = v
	return b
}

// Summary sets a one-line human description, overriding the generated one.
func (b *Builder) Summary(s string) *Builder {
	b.summary = s
	return b
}

// Has reports whether a finding with the given ID was recorded, letting later
// checks branch on the outcome of earlier ones.
func (b *Builder) Has(id string) bool {
	for _, f := range b.findings {
		if f.ID == id {
			return true
		}
	}
	return false
}

// Build computes the score, grade and verdict, sorts findings worst-first and
// stamps the elapsed time.
func (b *Builder) Build() *Report {
	score := 100
	verdict := StatusPass

	for _, f := range b.findings {
		w := severityWeight[f.Severity]
		switch f.Status {
		case StatusFail:
			score -= w
			verdict = StatusFail
		case StatusWarn:
			score -= w / 2
			if verdict != StatusFail {
				verdict = StatusWarn
			}
		}
	}
	if score < 0 {
		score = 0
	}

	findings := make([]Finding, len(b.findings))
	copy(findings, b.findings)
	sort.SliceStable(findings, func(i, j int) bool {
		if statusRank[findings[i].Status] != statusRank[findings[j].Status] {
			return statusRank[findings[i].Status] > statusRank[findings[j].Status]
		}
		return severityRank[findings[i].Severity] > severityRank[findings[j].Severity]
	})

	r := &Report{
		Tool:      b.tool,
		Target:    b.target,
		Verdict:   verdict,
		Score:     score,
		Grade:     grade(score),
		Summary:   b.summary,
		Findings:  findings,
		Raw:       b.raw,
		TimingMS:  time.Since(b.started).Milliseconds(),
		CheckedAt: time.Now().UTC(),
	}
	if r.Summary == "" {
		r.Summary = defaultSummary(r)
	}
	return r
}

// BuildError produces a report for a target that could not be inspected at
// all, so callers always get the same envelope back.
func (b *Builder) BuildError(id, title, detail string) *Report {
	b.findings = nil
	b.Fail(id, title, SeverityCritical, detail, "")
	r := b.Build()
	r.Score = 0
	r.Grade = grade(0)
	r.Summary = detail
	return r
}

func grade(score int) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

func defaultSummary(r *Report) string {
	pass, warn, fail := r.Counts()
	switch {
	case fail > 0:
		return plural(fail, "issue", "issues") + " found, " + plural(warn, "warning", "warnings") + ", " + plural(pass, "check passed", "checks passed")
	case warn > 0:
		return "No failures, " + plural(warn, "warning", "warnings") + ", " + plural(pass, "check passed", "checks passed")
	default:
		return "All " + plural(pass, "check passed", "checks passed")
	}
}

func plural(n int, one, many string) string {
	word := many
	if n == 1 {
		word = one
	}
	return itoa(n) + " " + word
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
