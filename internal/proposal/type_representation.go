package proposal

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func detectTypeRepresentationRepairs(records []*extract.Record, errs map[string][]rules.ValidationError) ([]Proposal, []TypeFinding) {
	recordsByPath := make(map[string]*extract.Record, len(records))
	for _, record := range records {
		recordsByPath[record.Path] = record
	}

	var proposals []Proposal
	var findings []TypeFinding
	for path, pathErrs := range errs {
		for _, validationErr := range pathErrs {
			if validationErr.Rule != "type" {
				continue
			}

			record := recordsByPath[path]
			scalar, hasScalar := extract.FrontmatterScalar{}, false
			if record != nil {
				scalar, hasScalar = record.FrontmatterScalars[validationErr.Field]
			}
			repairable := validationErr.ExpectedRepresentation == "string" &&
				extract.IsRepairableScalarRepresentation(validationErr.ActualRepresentation) &&
				hasScalar && scalar.Representation == validationErr.ActualRepresentation

			if !repairable {
				findings = append(findings, TypeFinding{
					Path:                 path,
					Field:                validationErr.Field,
					Message:              validationErr.Message,
					ActualRepresentation: validationErr.ActualRepresentation,
				})
				continue
			}

			proposals = append(proposals, Proposal{
				Type:               CorrectValue,
				Field:              validationErr.Field,
				Description:        fmt.Sprintf("quote %s value %q as an exact string", scalar.Representation, scalar.Lexeme),
				Paths:              []string{path},
				From:               scalar.Lexeme,
				To:                 scalar.Lexeme,
				FromRepresentation: scalar.Representation,
			})
		}
	}
	return proposals, findings
}
