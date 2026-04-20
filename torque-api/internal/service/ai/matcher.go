package ai

import (
	"strings"

	"github.com/google/uuid"
)

// TriggerRule is one row in agent_triggers reduced to the minimum the
// matcher needs. The repo is responsible for hydrating this from a
// live DB row — the matcher itself is pure (no DB round-trips) so it
// is trivially unit-testable with table-driven cases.
type TriggerRule struct {
	ID       uuid.UUID
	AgentID  uuid.UUID
	Priority int
	// IsActive is the soft-disable flag. The query that materialises
	// TriggerRule already filters WHERE is_active = true, but we keep
	// the field so callers composing in-memory scenarios can reason
	// about it without re-building the DB round trip.
	IsActive bool
	// AgentKillSwitch + AgentStatus come from the agents row joined at
	// query time. The matcher refuses to return an agent whose status
	// is not 'active' OR whose kill_switch is on — even if the filter
	// would otherwise match. This is the runtime gate that a flipped
	// kill switch takes effect on the very next assignment attempt.
	AgentKillSwitch bool
	AgentStatus     string
	Filter          FilterSpec
}

// FilterSpec is the JSON-backed DSL:
//
//	{
//	  "all": [ { "field": "origin", "op": "eq", "value": "meta-ads" } ],
//	  "any": [ { "field": "tags",   "op": "in", "value": ["vip","hot"] } ]
//	}
//
// Semantics:
//   - every predicate in `all` must pass (AND).
//   - at least one predicate in `any` must pass, if the list is non-empty.
//   - an entirely empty filter (`all` and `any` both empty) matches
//     every lead. That is the shape tenants use to build a "default"
//     catch-all rule at a high priority number.
type FilterSpec struct {
	All []Predicate `json:"all,omitempty"`
	Any []Predicate `json:"any,omitempty"`
}

// Predicate is a single comparison against a named lead attribute.
// Ops are deliberately small — complex tenant policies belong in a
// workflow (F07), not in a filter DSL that must stay readable by an
// account manager.
type Predicate struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// LeadFacts is the projection of domain.Lead the matcher operates on.
// Kept intentionally small — adding a new matchable attribute is a
// two-line change here + a repo update. Building on domain.Lead
// directly would entangle the matcher with every future lead field.
type LeadFacts struct {
	Origin      string
	Segment     string
	UTMSource   string
	UTMMedium   string
	UTMCampaign string
	Rating      *int16
	// Tags is a multi-value attribute. When the predicate uses op=in
	// against a string scalar, we check membership. When op=contains,
	// we expect `value` to be a string and verify it appears inside.
	Tags []string
	// Custom is a bag for arbitrary JSON attributes surfaced from
	// `leads.custom_fields` — the repo flattens top-level keys into
	// this map before calling the matcher.
	Custom map[string]any
}

// Match walks `rules` in the order they were provided (caller sorts
// by priority ASC then created_at ASC) and returns the first rule
// whose filter matches the lead. `found` is false when no rule
// matches.
//
// Kill-switch and status guards: a rule pointing at an inactive or
// kill-switched agent is skipped even if its filter matches. The
// matcher is deliberately silent about the reason — higher-priority
// logic can inspect the agents table if it needs to explain to the
// user why assignment didn't happen.
func Match(rules []TriggerRule, lead LeadFacts) (match TriggerRule, found bool) {
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		if rule.AgentKillSwitch || rule.AgentStatus != "active" {
			continue
		}
		if evalFilter(rule.Filter, lead) {
			return rule, true
		}
	}
	return TriggerRule{}, false
}

// evalFilter applies the AND/ANY semantics described on FilterSpec.
func evalFilter(f FilterSpec, lead LeadFacts) bool {
	// Empty filter matches everything — used by catch-all rules.
	if len(f.All) == 0 && len(f.Any) == 0 {
		return true
	}
	for _, p := range f.All {
		if !evalPredicate(p, lead) {
			return false
		}
	}
	if len(f.Any) > 0 {
		anyPass := false
		for _, p := range f.Any {
			if evalPredicate(p, lead) {
				anyPass = true
				break
			}
		}
		if !anyPass {
			return false
		}
	}
	return true
}

// evalPredicate runs one comparison. Unknown fields are treated as
// absent (matches IS NULL-style predicates but fails equality).
// Unknown operators default to false so a malformed rule can never
// surprise with a permissive true.
func evalPredicate(p Predicate, lead LeadFacts) bool {
	val := lookupField(p.Field, lead)
	switch strings.ToLower(p.Op) {
	case "eq":
		return toString(val) == toString(p.Value)
	case "neq":
		return toString(val) != toString(p.Value)
	case "contains":
		needle := toString(p.Value)
		if needle == "" {
			return false
		}
		if tags, ok := val.([]string); ok {
			for _, t := range tags {
				if strings.Contains(strings.ToLower(t), strings.ToLower(needle)) {
					return true
				}
			}
			return false
		}
		return strings.Contains(strings.ToLower(toString(val)), strings.ToLower(needle))
	case "in":
		list, ok := toStringSlice(p.Value)
		if !ok {
			return false
		}
		target := toString(val)
		// If val is itself a slice (e.g., tags), check intersection.
		if tags, ok2 := val.([]string); ok2 {
			for _, t := range tags {
				for _, v := range list {
					if strings.EqualFold(t, v) {
						return true
					}
				}
			}
			return false
		}
		for _, v := range list {
			if strings.EqualFold(target, v) {
				return true
			}
		}
		return false
	case "present":
		return !isEmpty(val)
	case "absent":
		return isEmpty(val)
	default:
		return false
	}
}

func lookupField(field string, lead LeadFacts) any {
	switch strings.ToLower(field) {
	case "origin":
		return lead.Origin
	case "segment":
		return lead.Segment
	case "utm_source":
		return lead.UTMSource
	case "utm_medium":
		return lead.UTMMedium
	case "utm_campaign":
		return lead.UTMCampaign
	case "rating":
		if lead.Rating == nil {
			return nil
		}
		return *lead.Rating
	case "tags":
		return lead.Tags
	default:
		if v, ok := lead.Custom[field]; ok {
			return v
		}
		return nil
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return intToString(int64(x))
	case int16:
		return intToString(int64(x))
	case int32:
		return intToString(int64(x))
	case int64:
		return intToString(x)
	case float64:
		// JSON numbers arrive as float64; if it's integer-valued, render
		// without decimals so `{op:eq, value: 5}` matches `rating: int16(5)`.
		if x == float64(int64(x)) {
			return intToString(int64(x))
		}
		return floatToString(x)
	default:
		return ""
	}
}

func intToString(n int64) string {
	// Minimal converter to avoid pulling fmt for a hot path.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}

// Intentionally minimal; unused path today but keeps toString total.
func floatToString(f float64) string {
	// Trivial fallback — the matcher does not compute against floats,
	// so we never hit this path in practice.
	return intToString(int64(f))
}

func toStringSlice(v any) ([]string, bool) {
	switch x := v.(type) {
	case []string:
		return x, true
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			out = append(out, toString(e))
		}
		return out, true
	default:
		return nil, false
	}
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []string:
		return len(x) == 0
	default:
		return false
	}
}
