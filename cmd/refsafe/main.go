package main

import (
	"github.com/Matts966/refsafe"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(refsafe.Analyzer) }
