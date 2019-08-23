package analysisutil_test

import (
	"testing"

	"github.com/Matts966/refsafe/analysisutil"
)

func TestCalledFrom(t *testing.T) {
	analysisutil.CalledFrom(nil, 5, nil, nil)
}
func TestCalledFromBefore(t *testing.T) {
	analysisutil.CalledFromBefore(nil, 5, nil, nil)
}
