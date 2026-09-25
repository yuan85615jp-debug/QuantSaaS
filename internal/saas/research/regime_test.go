package research

import "testing"

func TestAdviseOffenseOnHighFearLowCycle(t *testing.T) {
	fear := 85.0
	cycle := 15.0
	a := Advise(Input{FearScore: &fear, MarksCycle: &cycle})
	if a.Posture != PostureOffense {
		t.Fatalf("posture %s score %v", a.Posture, a.Score)
	}
}

func TestAdviseDefenseOnHighVIXHighCycle(t *testing.T) {
	vix := 35.0
	cycle := 85.0
	a := Advise(Input{VIX: &vix, MarksCycle: &cycle})
	if a.Posture != PostureDefense {
		t.Fatalf("posture %s score %v", a.Posture, a.Score)
	}
}

func TestAdviseNeutralNoInput(t *testing.T) {
	a := Advise(Input{})
	if a.Posture != PostureNeutral {
		t.Fatal(a.Posture)
	}
}
