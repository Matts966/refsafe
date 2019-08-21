package refsafe_test

import (
	"testing"

	"github.com/Matts966/refsafe"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, refsafe.Analyzer, "a")
}
