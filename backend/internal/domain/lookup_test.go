package domain

import (
	"testing"

	"devsignal/internal/validation"
)

func TestDefaultLookupsCoverEveryKind(t *testing.T) {
	l := DefaultLookups()
	for _, kind := range LookupKinds {
		if len(l[kind]) == 0 {
			t.Errorf("kind %q has no default options", kind)
		}
	}
	if len(l) != len(LookupKinds) {
		t.Errorf("defaults define %d kinds, LookupKinds lists %d", len(l), len(LookupKinds))
	}
}

func TestLookupsValues(t *testing.T) {
	l := Lookups{"k": {{Value: "b", Label: "B"}, {Value: "a", Label: "A"}}}
	got := l.Values("k")
	if len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Errorf("Values preserved order wrong: %v", got)
	}
	if v := l.Values("missing"); len(v) != 0 {
		t.Errorf("unknown kind should yield no values, got %v", v)
	}
}

// Every entity that carries a fixed-choice field must declare it, so the
// handler can validate it against the lookup table.
func TestEnumFieldsDeclared(t *testing.T) {
	cases := []struct {
		name   string
		entity Enumerated
		want   []EnumField
	}{
		{"application", &Application{Status: "applied", WorkMode: "remote"},
			[]EnumField{{"status", KindApplicationStatus, "applied"}, {"workMode", KindWorkMode, "remote"}}},
		{"contact", &Contact{Relationship: "peer"},
			[]EnumField{{"relationship", KindContactRelationship, "peer"}}},
		{"interview", &Interview{Stage: "final", Outcome: "passed"},
			[]EnumField{{"stage", KindInterviewStage, "final"}, {"outcome", KindInterviewOutcome, "passed"}}},
		{"skill", &Skill{Category: "cloud"},
			[]EnumField{{"category", KindSkillCategory, "cloud"}}},
		{"certification", &Certification{Status: "earned"},
			[]EnumField{{"status", KindCertificationStatus, "earned"}}},
		{"project", &Project{Status: "active"},
			[]EnumField{{"status", KindProjectStatus, "active"}}},
		{"goal", &Goal{Category: "skill", Status: "completed"},
			[]EnumField{{"category", KindGoalCategory, "skill"}, {"status", KindGoalStatus, "completed"}}},
	}
	for _, tc := range cases {
		got := tc.entity.EnumFields()
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s[%d]: got %+v, want %+v", tc.name, i, got[i], tc.want[i])
			}
		}
	}
}

func TestValidateEnums(t *testing.T) {
	l := DefaultLookups()

	e := validation.New()
	ValidateEnums(e, &Application{Status: "applied", WorkMode: ""}, l)
	if !e.Ok() {
		t.Errorf("valid status and empty optional work mode should pass, got %v", e)
	}

	e = validation.New()
	ValidateEnums(e, &Application{Status: "bogus", WorkMode: "moon"}, l)
	if e["status"] == "" || e["workMode"] == "" {
		t.Errorf("unknown values should be rejected per field, got %v", e)
	}

	// A kind missing from the lookup table rejects everything except empty,
	// so a misconfigured database fails closed rather than open.
	e = validation.New()
	ValidateEnums(e, &Application{Status: "applied"}, Lookups{})
	if e["status"] == "" {
		t.Errorf("missing kind should reject non-empty values, got %v", e)
	}
}
