package ictrp

import (
	"encoding/xml"
	"testing"
)

func TestParseICTRPXML(t *testing.T) {
	body := []byte(`<?xml version="1.0"?>
<Trials_downloaded_from_ICTRP>
  <Trial>
    <TrialID>EUCTR2024-500001-12-DE</TrialID>
    <Source_Register>EU CTIS</Source_Register>
    <Recruitment_Status>Recruiting</Recruitment_Status>
    <Phase>Phase 3</Phase>
    <Public_title>Sample Trial</Public_title>
  </Trial>
  <Trial>
    <TrialID>ChiCTR2400000001</TrialID>
    <Source_Register>ChiCTR</Source_Register>
  </Trial>
</Trials_downloaded_from_ICTRP>`)
	var root ictrpRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		t.Fatal(err)
	}
	if len(root.Trials) != 2 {
		t.Fatalf("want 2 trials, got %d", len(root.Trials))
	}
	if root.Trials[0].TrialID != "EUCTR2024-500001-12-DE" {
		t.Errorf("first id: %q", root.Trials[0].TrialID)
	}
}
