package ai_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/service/ai"
)

func mkRule(id, agentID uuid.UUID, priority int, filter ai.FilterSpec) ai.TriggerRule {
	return ai.TriggerRule{
		ID: id, AgentID: agentID, Priority: priority, IsActive: true,
		AgentKillSwitch: false, AgentStatus: "active",
		Filter: filter,
	}
}

// --------------- empty filter + no rules ---------------------------

func TestMatcher_NoRulesReturnsNotFound(t *testing.T) {
	t.Parallel()
	_, found := ai.Match(nil, ai.LeadFacts{})
	if found {
		t.Errorf("want no match for empty rule set")
	}
}

func TestMatcher_EmptyFilterMatchesEverything(t *testing.T) {
	t.Parallel()
	a := uuid.New()
	rules := []ai.TriggerRule{mkRule(uuid.New(), a, 100, ai.FilterSpec{})}
	r, found := ai.Match(rules, ai.LeadFacts{Origin: "organic"})
	if !found {
		t.Fatal("expected match for empty filter")
	}
	if r.AgentID != a {
		t.Errorf("wrong agent matched")
	}
}

// --------------- kill switch / status guards -----------------------

func TestMatcher_KillSwitchRuleSkipped(t *testing.T) {
	t.Parallel()
	r := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{})
	r.AgentKillSwitch = true
	_, found := ai.Match([]ai.TriggerRule{r}, ai.LeadFacts{})
	if found {
		t.Errorf("kill-switch should block the match")
	}
}

func TestMatcher_NonActiveAgentSkipped(t *testing.T) {
	t.Parallel()
	r := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{})
	r.AgentStatus = "disabled"
	_, found := ai.Match([]ai.TriggerRule{r}, ai.LeadFacts{})
	if found {
		t.Errorf("disabled agent should block the match")
	}
}

func TestMatcher_InactiveRuleSkipped(t *testing.T) {
	t.Parallel()
	r := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{})
	r.IsActive = false
	_, found := ai.Match([]ai.TriggerRule{r}, ai.LeadFacts{})
	if found {
		t.Errorf("inactive rule should be skipped")
	}
}

// --------------- priority ordering ---------------------------------

func TestMatcher_LowerPriorityWins(t *testing.T) {
	t.Parallel()
	wanted := uuid.New()
	loser := uuid.New()
	rules := []ai.TriggerRule{
		mkRule(uuid.New(), wanted, 10, ai.FilterSpec{}),
		mkRule(uuid.New(), loser, 100, ai.FilterSpec{}),
	}
	r, _ := ai.Match(rules, ai.LeadFacts{})
	if r.AgentID != wanted {
		t.Errorf("want priority 10 to beat 100, got %v", r.AgentID)
	}
}

// --------------- eq / neq ------------------------------------------

func TestMatcher_EqMatchesStringField(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "origin", Op: "eq", Value: "meta-ads"}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Origin: "meta-ads"})
	if !found {
		t.Errorf("eq missed matching lead")
	}
	_, found2 := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Origin: "organic"})
	if found2 {
		t.Errorf("eq matched wrong value")
	}
}

func TestMatcher_NeqFlipsEq(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "origin", Op: "neq", Value: "organic"}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Origin: "meta-ads"})
	if !found {
		t.Errorf("neq should match when values differ")
	}
}

// --------------- contains ------------------------------------------

func TestMatcher_ContainsMatchesTagsSlice(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "tags", Op: "contains", Value: "hot"}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Tags: []string{"vip", "hot-lead"}})
	if !found {
		t.Errorf("contains failed to match tag substring")
	}
}

// --------------- in ------------------------------------------------

func TestMatcher_InMatchesStringInList(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "segment", Op: "in", Value: []string{"enterprise", "mid-market"}}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Segment: "enterprise"})
	if !found {
		t.Errorf("in-list match failed")
	}
}

func TestMatcher_InMatchesTagsIntersection(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "tags", Op: "in", Value: []string{"vip", "returning"}}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Tags: []string{"hot", "vip"}})
	if !found {
		t.Errorf("intersection in list failed")
	}
}

// --------------- present / absent ---------------------------------

func TestMatcher_PresentPassesWhenValueNonEmpty(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "utm_campaign", Op: "present"}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{UTMCampaign: "spring-sale"})
	if !found {
		t.Errorf("present failed on non-empty")
	}
	_, found2 := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{})
	if found2 {
		t.Errorf("present matched empty")
	}
}

func TestMatcher_AbsentPassesOnEmpty(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "origin", Op: "absent"}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{})
	if !found {
		t.Errorf("absent failed on empty")
	}
}

// --------------- all+any composition ------------------------------

func TestMatcher_AllAndAnyCombined(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "origin", Op: "eq", Value: "meta-ads"}},
		Any: []ai.Predicate{
			{Field: "segment", Op: "eq", Value: "enterprise"},
			{Field: "tags", Op: "contains", Value: "vip"},
		},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{
		Origin: "meta-ads", Segment: "smb", Tags: []string{"vip"},
	})
	if !found {
		t.Errorf("all+any should match when all passes and at least one any passes")
	}
	// Same lead but no any predicate passes → no match.
	_, found2 := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{
		Origin: "meta-ads", Segment: "smb", Tags: []string{"cold"},
	})
	if found2 {
		t.Errorf("all+any passed when any block failed")
	}
}

func TestMatcher_UnknownOperatorFailsClosed(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "origin", Op: "regex", Value: ".*"}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Origin: "anything"})
	if found {
		t.Errorf("unknown op must default to false")
	}
}

// --------------- custom field lookup ------------------------------

func TestMatcher_CustomFieldEq(t *testing.T) {
	t.Parallel()
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "plan", Op: "eq", Value: "pro"}},
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{
		Custom: map[string]any{"plan": "pro"},
	})
	if !found {
		t.Errorf("custom field eq failed")
	}
}

// --------------- rating numeric round-trip ------------------------

func TestMatcher_RatingInt16EqNumber(t *testing.T) {
	t.Parallel()
	r := int16(5)
	rule := mkRule(uuid.New(), uuid.New(), 100, ai.FilterSpec{
		All: []ai.Predicate{{Field: "rating", Op: "eq", Value: float64(5)}}, // JSON -> float64
	})
	_, found := ai.Match([]ai.TriggerRule{rule}, ai.LeadFacts{Rating: &r})
	if !found {
		t.Errorf("rating eq via float64 failed")
	}
}
