package main

import (
	"context"
	"fmt"
	"gestor-documental/internal/buildinfo"
	"gestor-documental/internal/config"
)

func doctor(ctx context.Context, configuration config.Config, samples string) error {
	fmt.Printf("AIBID %s (%s)\nConfiguración: OK\n", buildinfo.Version, buildinfo.Channel)
	results, err := configuration.Indexing.Diagnose(ctx)
	for _, result := range results {
		fmt.Println(result)
	}
	if err != nil {
		return err
	}
	if samples != "" {
		if err := configuration.Indexing.DiagnoseSamples(ctx, samples); err != nil {
			return err
		}
		fmt.Println("PDF nativo y OCR español: OK")
	}
	return nil
}
