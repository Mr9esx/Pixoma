package domain_test

import (
	"testing"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestSkillVersionUsesThreeCanonicalNumericParts(t *testing.T) {
	for _, version := range []string{"0.0.0", "1.2.3", "999999999999999999999.0.1"} {
		if !domain.ValidSkillVersion(version) {
			t.Errorf("valid version rejected: %q", version)
		}
	}
	for _, version := range []string{"", "1", "1.2", "1.2.3.4", "01.2.3", "1.02.3", "1.2.03", "v1.2.3", "1.2.3-rc.1", "1.2.3+build.1", "-1.2.3", "1.2.3\n"} {
		if domain.ValidSkillVersion(version) {
			t.Errorf("invalid version accepted: %q", version)
		}
	}
	if domain.CompareSkillVersions("1.0.10", "1.0.9") <= 0 ||
		domain.CompareSkillVersions("10.0.0", "2.999.999") <= 0 ||
		domain.CompareSkillVersions("999999999999999999999.0.1", "9.999.999") <= 0 ||
		domain.CompareSkillVersions("1.0.9", "1.0.10") >= 0 ||
		domain.CompareSkillVersions("1.2.3", "1.2.3") != 0 {
		t.Fatal("numeric version ordering is incorrect")
	}
}
