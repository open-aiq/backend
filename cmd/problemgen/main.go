package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"go-aiq-backend/internal/platform/problem"
)

type documentedProblem struct {
	problem.Kind
	Type string `json:"type"`
}

func main() {
	output := flag.String("out", "docs/problems.json", "output path")
	flag.Parse()

	catalog := problem.Catalog()
	documented := make([]documentedProblem, 0, len(catalog))
	for _, kind := range catalog {
		documented = append(documented, documentedProblem{Kind: kind, Type: problem.TypeURL(kind)})
	}
	data, err := json.MarshalIndent(documented, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write problem catalog: %v\n", err)
		os.Exit(1)
	}
}
