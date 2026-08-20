package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

type prospectiveRecordValidationInput struct {
	Path                         string
	AbsPath                      string
	Content                      []byte
	ReadFile                     bool
	Effective                    *rules.StemFile
	ResolveEffective             func() (*rules.StemFile, error)
	LinkCache                    *rules.HeadingCache
	ProspectiveLinkTargetAbsPath string
	ProspectiveLinkTargetContent []byte
}

func validateProspectiveRecord(ctx context.Context, in prospectiveRecordValidationInput) (*rules.ValidationResult, error) {
	ext := extract.NewASTRegistry().ForFile(in.AbsPath, "")
	if ext == nil {
		return nil, fmt.Errorf("no extractor for %s", in.Path)
	}

	content := in.Content
	if in.ReadFile {
		var err error
		content, err = os.ReadFile(in.AbsPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", in.Path, err)
		}
	}

	record, err := extractProspectiveRecordWithExtractor(ext, in.Path, content)
	if err != nil {
		return nil, err
	}

	effective := in.Effective
	if effective == nil && in.ResolveEffective != nil {
		var err error
		effective, err = in.ResolveEffective()
		if err != nil {
			return nil, err
		}
	}

	linkCache := in.LinkCache
	if linkCache == nil {
		linkCache = rules.NewHeadingCache()
	}

	errs := rules.Validate(ctx, record, effective)
	if in.ProspectiveLinkTargetAbsPath != "" {
		errs = append(errs, rules.CheckLinksWithProspectiveTarget(record.Links, effective.Links, in.AbsPath, rules.SchemaRoot(in.AbsPath), linkCache, rules.ProspectiveLinkTarget{
			AbsPath: in.ProspectiveLinkTargetAbsPath,
			Content: in.ProspectiveLinkTargetContent,
		})...)
	} else {
		errs = append(errs, rules.CheckLinks(record.Links, effective.Links, in.AbsPath, rules.SchemaRoot(in.AbsPath), linkCache)...)
	}
	errs = append(errs, rules.ExtractionErrors(record)...)
	errs = append(errs, rules.ValidateStructure(content, in.Path)...)

	return rules.NewValidationResult(in.Path, errs), nil
}

func extractProspectiveRecord(absPath, recordPath string, content []byte) (*extract.Record, error) {
	ext := extract.NewASTRegistry().ForFile(absPath, "")
	if ext == nil {
		return nil, fmt.Errorf("no extractor for %s", recordPath)
	}
	return extractProspectiveRecordWithExtractor(ext, recordPath, content)
}

func extractProspectiveRecordWithExtractor(ext extract.Extractor, recordPath string, content []byte) (*extract.Record, error) {
	record, err := ext.Extract(recordPath, content)
	if err != nil {
		return nil, fmt.Errorf("extracting %s: %w", recordPath, err)
	}
	return record, nil
}

func validationResultHasFailure(result *rules.ValidationResult, strict bool) bool {
	return validationBatchHasFailure(rules.NewValidationEnvelope(rules.ValidationEnvelopeInput{
		Results: []*rules.ValidationResult{result},
	}), strict)
}
