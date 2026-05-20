package infer

import "testing"

func TestDetectMissingDomains_Deprecated(t *testing.T) {
	if got := DetectMissingDomains(nil); got != nil {
		t.Errorf("DetectMissingDomains(nil) = %v, want nil", got)
	}
}
