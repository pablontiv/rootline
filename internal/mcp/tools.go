package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
)

// RegisterTools registers all rootline MCP tools on the server.
func RegisterTools(s *Server) {
	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "query",
		Description: "Search and filter records by frontmatter fields using expr-lang expressions",
	}, handleQuery)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "validate",
		Description: "Check documents against .stem schema rules",
	}, handleValidate)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "describe",
		Description: "Show the effective .stem schema for a directory",
	}, handleDescribe)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "tree",
		Description: "Show hierarchical tree of records with completion counts",
	}, handleTree)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "stats",
		Description: "Show aggregate statistics (by estado, by tipo) for records",
	}, handleStats)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "explain",
		Description: "Trace why a document has a given state: field origins, derivation chain, and validation errors",
	}, handleExplain)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "fix",
		Description: "Analyze validation errors and return fix proposals (always dry-run, never modifies files)",
	}, handleFix)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "graph",
		Description: "Build dependency graph from wiki-links with cycle detection and broken link analysis",
	}, handleGraph)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "set",
		Description: "Set field values on a document (frontmatter or body sections). Validates against .stem schema.",
	}, handleSet)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "trace",
		Description: "Follow reference chains through the document graph via BFS traversal. Shows connected documents and their estado.",
	}, handleTrace)
}

// Tool input types

// QueryInput is the input for the query tool.
type QueryInput struct {
	Path  string   `json:"path" jsonschema:"directory to scan (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions (expr-lang syntax)"`
	Count bool     `json:"count,omitempty" jsonschema:"return count instead of records"`
	Limit int      `json:"limit,omitempty" jsonschema:"limit number of results (0 = unlimited)"`
}

// ValidateInput is the input for the validate tool.
type ValidateInput struct {
	Path  string   `json:"path" jsonschema:"directory to validate (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions for validation scope"`
}

// DescribeInput is the input for the describe tool.
type DescribeInput struct {
	Path   string `json:"path" jsonschema:"directory to describe (absolute path)"`
	Domain string `json:"domain,omitempty" jsonschema:"filter output to fields matching this domain (e.g. lifecycle_state)"`
}

// TreeInput is the input for the tree tool.
type TreeInput struct {
	Path  string   `json:"path" jsonschema:"directory to scan (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions"`
}

// StatsInput is the input for the stats tool.
type StatsInput struct {
	Path  string   `json:"path" jsonschema:"directory to scan (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions"`
}

// ExplainInput is the input for the explain tool.
type ExplainInput struct {
	Path string `json:"path" jsonschema:"file to explain (absolute path)"`
}

// FixInput is the input for the fix tool.
type FixInput struct {
	Path   string `json:"path" jsonschema:"directory to scan (absolute path)"`
	DryRun bool   `json:"dry_run,omitempty" jsonschema:"always true for MCP (proposals only)"`
	All    bool   `json:"all,omitempty" jsonschema:"fix all files in scope"`
}

// GraphInput is the input for the graph tool.
type GraphInput struct {
	Path   string `json:"path" jsonschema:"directory to scan (absolute path)"`
	Check  bool   `json:"check,omitempty" jsonschema:"validate only (cycles + broken links)"`
	Format string `json:"format,omitempty" jsonschema:"output format: dot or mermaid (default: json)"`
}

// TraceInput is the input for the trace tool.
type TraceInput struct {
	Path     string `json:"path" jsonschema:"absolute path to the starting file"`
	Reverse  bool   `json:"reverse,omitempty" jsonschema:"follow incoming references instead of outgoing"`
	Depth    int    `json:"depth,omitempty" jsonschema:"max traversal depth (0 = unlimited)"`
	EdgeType string `json:"edge_type,omitempty" jsonschema:"filter by edge type (field name)"`
}

// SetInput is the input for the set tool.
type SetInput struct {
	Path           string            `json:"path" jsonschema:"absolute path to document file"`
	Fields         map[string]string `json:"fields" jsonschema:"field name to value mapping (replaces existing)"`
	AppendFields   map[string]string `json:"append_fields,omitempty" jsonschema:"fields to append content to (section fields only)"`
	CreateSections bool              `json:"create_sections,omitempty" jsonschema:"create body sections that do not exist"`
	DryRun         bool              `json:"dry_run,omitempty" jsonschema:"show changes without applying"`
}

func handleQuery(ctx context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	q := &query.Query{Count: input.Count, Limit: input.Limit}
	result, err := query.ExecuteExpr(ctx, filtered, "", q)
	if err != nil {
		return nil, nil, fmt.Errorf("executing query: %w", err)
	}

	return jsonResult(result)
}

func handleValidate(ctx context.Context, _ *mcp.CallToolRequest, input ValidateInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	var results []*rules.ValidationResult
	for _, rec := range filtered {
		absPath := filepath.Join(absRoot, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		errs := rules.Validate(ctx, rec, effective)
		results = append(results, rules.NewValidationResult(rec.Path, errs))
	}

	batch := rules.NewBatchValidationResult(results)
	return jsonResult(batch)
}

func handleDescribe(ctx context.Context, _ *mcp.CallToolRequest, input DescribeInput) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	entries, err := rules.WalkUp(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("walking up: %w", err)
	}

	effective := rules.MergeStemFiles(entries)
	result := rules.NewDescribeResult(input.Path, entries, effective)

	// Filter by domain if requested
	if input.Domain != "" {
		filtered := make(map[string]rules.SchemaField)
		for name, field := range result.Schema {
			if field.Domain == input.Domain {
				filtered[name] = field
			}
		}
		result.Schema = filtered
	}

	return jsonResult(result)
}

// treeNode represents a directory or file in the tree.
type treeNode struct {
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Children  []*treeNode `json:"children,omitempty"`
	Completed int         `json:"completed"`
	Total     int         `json:"total"`
	IsLeaf    bool        `json:"is_leaf,omitempty"`
	Estado    string      `json:"estado,omitempty"`
}

// treeResult is the versioned JSON output for tree.
type treeResult struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Root    *treeNode `json:"root"`
}

func handleTree(ctx context.Context, _ *mcp.CallToolRequest, input TreeInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	// Resolve lifecycle field via domain lookup with fallback
	estadoField := "estado"
	stemEntries, walkErr := rules.WalkUp(absRoot)
	if walkErr == nil {
		if eff := rules.MergeStemFiles(stemEntries); eff != nil {
			if name, found := rules.FindFieldByDomain(eff.Schema, "lifecycle_state", absRoot); found {
				estadoField = name
			}
		}
	}

	root := buildTree(filtered, filepath.Base(absRoot), estadoField)
	result := &treeResult{Version: 1, Kind: "rootline/tree", Root: root}
	return jsonResult(result)
}

// statsResult is the versioned JSON output for stats.
type statsResult struct {
	Version  int            `json:"version"`
	Kind     string         `json:"kind"`
	ByEstado map[string]int `json:"by_estado"`
	ByTipo   map[string]int `json:"by_tipo"`
	Total    int            `json:"total"`
}

func handleStats(ctx context.Context, _ *mcp.CallToolRequest, input StatsInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	// Resolve field names via domain lookup with fallback
	estadoField := "estado"
	tipoField := "tipo"
	entries, walkErr := rules.WalkUp(absRoot)
	if walkErr == nil {
		if eff := rules.MergeStemFiles(entries); eff != nil {
			if name, found := rules.FindFieldByDomain(eff.Schema, "lifecycle_state", absRoot); found {
				estadoField = name
			}
			if name, found := rules.FindFieldByDomain(eff.Schema, "record_type", absRoot); found {
				tipoField = name
			}
		}
	}

	byEstado := make(map[string]int)
	byTipo := make(map[string]int)
	for _, rec := range filtered {
		if v, ok := rec.EffectiveField(estadoField); ok {
			byEstado[fmt.Sprintf("%v", v)]++
		}
		if v, ok := rec.EffectiveField(tipoField); ok {
			byTipo[fmt.Sprintf("%v", v)]++
		}
	}

	result := &statsResult{
		Version:  1,
		Kind:     "rootline/stats",
		ByEstado: byEstado,
		ByTipo:   byTipo,
		Total:    len(filtered),
	}
	return jsonResult(result)
}

func handleExplain(ctx context.Context, _ *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, nil, fmt.Errorf("file not found: %s", input.Path)
	}

	reg := extract.NewRegistry()
	ext := reg.ForFile(absPath, "")
	if ext == nil {
		return nil, nil, fmt.Errorf("no extractor for %s", input.Path)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", input.Path, err)
	}

	record, err := ext.Extract(input.Path, content)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting %s: %w", input.Path, err)
	}

	dir := filepath.Dir(absPath)
	entries, err := rules.WalkUp(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving .stem for %s: %w", input.Path, err)
	}
	effective := rules.MergeStemFiles(entries)

	if effective != nil && len(effective.Derive) > 0 {
		_, _ = derive.DeriveRecord(ctx, record, effective, nil)
	}

	if effective != nil && len(effective.Aggregate) > 0 && filepath.Base(absPath) == "README.md" {
		scanDir := filepath.Dir(absPath)
		reg2 := extract.NewRegistry()
		siblings, scanErr := index.Scan(ctx, scanDir, reg2)
		if scanErr == nil && len(siblings) > 0 {
			derive.DeriveAllSimple(ctx, siblings, scanDir)
			derive.AggregateAllSimple(ctx, siblings, scanDir)
			for _, s := range siblings {
				sAbs := filepath.Join(scanDir, s.Path)
				if sAbs == absPath {
					record.Derived = s.Derived
					break
				}
			}
		}
	}

	var valErrs []rules.ValidationError
	if effective != nil {
		valErrs = rules.Validate(ctx, record, effective)
	}

	result := rules.NewExplainResult(input.Path, entries, effective, record, valErrs)
	return jsonResult(result)
}

func handleFix(ctx context.Context, _ *mcp.CallToolRequest, input FixInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	allErrs := make(map[string][]rules.ValidationError)
	effectiveStems := make(map[string]*rules.StemFile)

	for _, rec := range records {
		recAbsPath := filepath.Join(absRoot, rec.Path)
		dir := filepath.Dir(recAbsPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		effectiveStems[rec.Path] = effective
		errs := rules.Validate(ctx, rec, effective)
		if len(errs) > 0 {
			allErrs[rec.Path] = errs
		}
	}

	var effective *rules.StemFile
	for _, s := range effectiveStems {
		if effective == nil || len(s.Schema) > len(effective.Schema) {
			effective = s
		}
	}

	report := proposal.Analyze(records, effective, allErrs)
	return jsonResult(report)
}

// graphResult is the versioned JSON output for graph.
type graphResult struct {
	Version     int                `json:"version"`
	Kind        string             `json:"kind"`
	Nodes       []string           `json:"nodes"`
	Edges       []graph.Edge       `json:"edges"`
	Cycles      [][]string         `json:"cycles"`
	BrokenLinks []graph.BrokenLink `json:"broken_links"`
}

func handleGraph(ctx context.Context, _ *mcp.CallToolRequest, input GraphInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	g := graph.Build(ctx, records)
	cycles := g.DetectCycles()
	broken := g.BrokenLinks()

	nodes := make([]string, 0, len(g.Nodes))
	for path := range g.Nodes {
		nodes = append(nodes, path)
	}
	sort.Strings(nodes)

	var allEdges []graph.Edge
	for _, edges := range g.Edges {
		allEdges = append(allEdges, edges...)
	}
	if allEdges == nil {
		allEdges = []graph.Edge{}
	}
	if broken == nil {
		broken = []graph.BrokenLink{}
	}
	if cycles == nil {
		cycles = [][]string{}
	}

	if input.Format == "dot" || input.Format == "mermaid" {
		var sb strings.Builder
		if input.Format == "dot" {
			sb.WriteString("digraph {\n  rankdir=LR;\n")
			for _, path := range nodes {
				fmt.Fprintf(&sb, "  %q;\n", path)
			}
			for _, e := range allEdges {
				fmt.Fprintf(&sb, "  %q -> %q [label=%q];\n", e.Source, e.Target, e.Type)
			}
			sb.WriteString("}\n")
		} else {
			sb.WriteString("graph TD;\n")
			replacer := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_")
			for _, path := range nodes {
				fmt.Fprintf(&sb, "  %s[%q];\n", replacer.Replace(path), path)
			}
			for _, e := range allEdges {
				fmt.Fprintf(&sb, "  %s --> |%s| %s;\n", replacer.Replace(e.Source), e.Type, replacer.Replace(e.Target))
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		}, nil, nil
	}

	result := &graphResult{
		Version:     1,
		Kind:        "rootline/graph",
		Nodes:       nodes,
		Edges:       allEdges,
		Cycles:      cycles,
		BrokenLinks: broken,
	}
	return jsonResult(result)
}

// setResult is the versioned JSON output for the set tool.
type setResult struct {
	Version  int      `json:"version"`
	Kind     string   `json:"kind"`
	Applied  []string `json:"applied"`
	DryRun   bool     `json:"dry_run,omitempty"`
	Previews []string `json:"previews,omitempty"`
}

func handleSet(ctx context.Context, _ *mcp.CallToolRequest, input SetInput) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, nil, fmt.Errorf("file not found: %s", input.Path)
	}

	// Read file.
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", input.Path, err)
	}

	// Find git root for schema resolution and relative path computation.
	dir := filepath.Dir(absPath)
	gitRoot, err := findGitRoot(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("finding git root: %w", err)
	}

	relPath, err := filepath.Rel(gitRoot, absPath)
	if err != nil {
		relPath = absPath
	}

	// Extract record with AST enabled (required for section mutations).
	parseAST := true
	ext := &extract.MarkdownExtractor{ParseAST: &parseAST}
	record, err := ext.Extract(relPath, content)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting %s: %w", input.Path, err)
	}

	// Load effective schema.
	effective, err := rules.ResolveForRecord(dir, absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving .stem for %s: %w", input.Path, err)
	}

	// Build fieldOps from Fields (replace) and AppendFields (append).
	type fieldOp struct {
		Field  string
		Value  string
		Append bool
	}
	var ops []fieldOp
	// Stable iteration order: sort keys.
	fieldKeys := make([]string, 0, len(input.Fields))
	for k := range input.Fields {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)
	for _, k := range fieldKeys {
		ops = append(ops, fieldOp{Field: k, Value: input.Fields[k], Append: false})
	}
	appendKeys := make([]string, 0, len(input.AppendFields))
	for k := range input.AppendFields {
		appendKeys = append(appendKeys, k)
	}
	sort.Strings(appendKeys)
	for _, k := range appendKeys {
		ops = append(ops, fieldOp{Field: k, Value: input.AppendFields[k], Append: true})
	}

	if len(ops) == 0 {
		return nil, nil, fmt.Errorf("no fields specified: provide fields or append_fields")
	}

	// Build proposals.
	var proposals []proposal.Proposal
	for _, op := range ops {
		p, pErr := buildSetProposal(op.Field, op.Value, op.Append, relPath, effective, input.CreateSections)
		if pErr != nil {
			return nil, nil, pErr
		}
		proposals = append(proposals, p)
	}

	// Pre-validate enum constraints (same as CLI set command).
	if effective != nil {
		for _, op := range ops {
			if op.Append {
				continue
			}
			sf, ok := effective.Schema[op.Field]
			if !ok {
				continue
			}
			if sf.Type == "section" {
				continue
			}
			if sf.Type == "enum" && len(sf.Values) > 0 && op.Value != "" {
				valid := false
				for _, v := range sf.Values {
					if v == op.Value {
						valid = true
						break
					}
				}
				if !valid {
					return nil, nil, fmt.Errorf("value %q not in allowed values for %s: [%s]",
						op.Value, op.Field, strings.Join(sf.Values, ", "))
				}
			}
		}
	}

	// Dry-run mode: return previews without modifying files.
	if input.DryRun {
		var previews []string
		for _, p := range proposals {
			switch p.Type {
			case proposal.SetField:
				previews = append(previews, fmt.Sprintf("would set %s = %q", p.Field, p.Value))
			case proposal.SetSection:
				mode := p.Mode
				if mode == "" {
					mode = "replace"
				}
				previews = append(previews, fmt.Sprintf("would %s section %s (%s)", mode, p.Heading, p.Field))
			}
		}
		result := &setResult{
			Version:  1,
			Kind:     "rootline/set",
			DryRun:   true,
			Previews: previews,
		}
		return jsonResult(result)
	}

	// Save original content for rollback.
	originalContent := make([]byte, len(content))
	copy(originalContent, content)

	// Apply proposals via fix.ApplyProposals.
	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	}
	recordMap := []*extract.Record{record}
	if err := fix.ApplyProposals(ctx, report, gitRoot, recordMap); err != nil {
		return nil, nil, fmt.Errorf("applying changes: %w", err)
	}

	// Post-validate: re-read, re-extract, validate; rollback on failure.
	newContent, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("re-reading %s after mutation: %w", input.Path, err)
	}
	newRecord, err := ext.Extract(absPath, newContent)
	if err != nil {
		_ = os.WriteFile(absPath, originalContent, 0o644)
		return nil, nil, fmt.Errorf("re-extracting %s after mutation (rolled back): %w", input.Path, err)
	}
	freshEffective, freshErr := rules.ResolveForRecord(dir, absPath)
	if freshErr != nil {
		freshEffective = effective
	}
	if freshEffective != nil {
		valErrs := rules.Validate(ctx, newRecord, freshEffective)
		if len(valErrs) > 0 {
			_ = os.WriteFile(absPath, originalContent, 0o644)
			var msgs []string
			for _, e := range valErrs {
				msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field, e.Message))
			}
			return nil, nil, fmt.Errorf("post-mutation validation failed (rolled back): %s", strings.Join(msgs, "; "))
		}
	}

	// Build summary of applied changes.
	var applied []string
	for _, p := range report.Proposals {
		switch p.Type {
		case proposal.SetField:
			applied = append(applied, fmt.Sprintf("set %s = %q", p.Field, p.Value))
		case proposal.SetSection:
			mode := p.Mode
			if mode == "" {
				mode = "replace"
			}
			applied = append(applied, fmt.Sprintf("%s section %s", mode, p.Heading))
		}
	}

	result := &setResult{
		Version: 1,
		Kind:    "rootline/set",
		Applied: applied,
	}
	return jsonResult(result)
}

// buildSetProposal creates a Proposal for the set tool, consulting the schema to determine type.
func buildSetProposal(field, value string, appendMode bool, relPath string, effective *rules.StemFile, createSections bool) (proposal.Proposal, error) {
	if effective != nil {
		if sf, ok := effective.Schema[field]; ok && sf.Type == "section" {
			heading := sf.Heading
			if heading == "" {
				heading = "## " + field
			}
			mode := "replace"
			if appendMode {
				mode = "append"
			} else if createSections {
				mode = "create"
			}
			return proposal.Proposal{
				Type:    proposal.SetSection,
				Field:   field,
				Heading: heading,
				Value:   value,
				Mode:    mode,
				Paths:   []string{relPath},
			}, nil
		}
	}

	if appendMode {
		return proposal.Proposal{}, fmt.Errorf("append is only supported for section fields; %q is not a section", field)
	}

	return proposal.Proposal{
		Type:        proposal.SetField,
		Field:       field,
		Value:       value,
		Paths:       []string{relPath},
		Description: fmt.Sprintf("set %s to %q", field, value),
	}, nil
}

// findGitRoot walks up from dir looking for a .git directory.
func findGitRoot(dir string) (string, error) {
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no .git directory found above %s", dir)
		}
		current = parent
	}
}

// Helpers

func filterWhere(ctx context.Context, records []*extract.Record, wheres []string) ([]*extract.Record, error) {
	var cleaned []string
	for _, w := range wheres {
		if w != "" {
			cleaned = append(cleaned, w)
		}
	}
	if len(cleaned) == 0 {
		return records, nil
	}

	combined := strings.Join(cleaned, " && ")
	program, err := query.CompileWhere(combined)
	if err != nil {
		return nil, fmt.Errorf("compiling where: %w", err)
	}

	var filtered []*extract.Record
	for _, rec := range records {
		match, matchErr := query.MatchRecord(ctx, program, rec, nil)
		if matchErr != nil {
			continue
		}
		if match {
			filtered = append(filtered, rec)
		}
	}
	return filtered, nil
}

func handleTrace(ctx context.Context, _ *mcp.CallToolRequest, input TraceInput) (*mcp.CallToolResult, any, error) {
	absFile, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	// Walk up to find .git root
	absRoot := filepath.Dir(absFile)
	for {
		if _, err := os.Stat(filepath.Join(absRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(absRoot)
		if parent == absRoot {
			return nil, nil, fmt.Errorf("no .git directory found above %s", input.Path)
		}
		absRoot = parent
	}

	relFile, err := filepath.Rel(absRoot, absFile)
	if err != nil {
		return nil, nil, fmt.Errorf("computing relative path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.EnrichBuiltinsSimple(ctx, records, absRoot)
	g := graph.Build(ctx, records)

	result := g.Trace(relFile, graph.TraceOptions{
		Reverse:  input.Reverse,
		MaxDepth: input.Depth,
		EdgeType: input.EdgeType,
	})

	return jsonResult(result)
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func buildTree(records []*extract.Record, rootName string, estadoField string) *treeNode {
	root := &treeNode{Name: rootName, Path: rootName}

	for _, rec := range records {
		parts := strings.Split(filepath.ToSlash(rec.Path), "/")
		node := root

		for i := 0; i < len(parts)-1; i++ {
			child := findChild(node, parts[i])
			if child == nil {
				child = &treeNode{Name: parts[i], Path: strings.Join(parts[:i+1], "/")}
				node.Children = append(node.Children, child)
			}
			node = child
		}

		estado := ""
		if e, ok := rec.EffectiveField(estadoField); ok {
			estado = fmt.Sprintf("%v", e)
		}
		leaf := &treeNode{
			Name:   parts[len(parts)-1],
			Path:   rec.Path,
			IsLeaf: true,
			Estado: estado,
		}
		if estado == "Completed" {
			leaf.Completed = 1
		}
		leaf.Total = 1
		node.Children = append(node.Children, leaf)
	}

	propagateCounts(root)
	return root
}

func findChild(node *treeNode, name string) *treeNode {
	for _, c := range node.Children {
		if c.Name == name && !c.IsLeaf {
			return c
		}
	}
	return nil
}

func propagateCounts(node *treeNode) {
	if node.IsLeaf {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Name < node.Children[j].Name
	})

	node.Completed = 0
	node.Total = 0
	for _, child := range node.Children {
		propagateCounts(child)
		node.Completed += child.Completed
		node.Total += child.Total
	}
}
