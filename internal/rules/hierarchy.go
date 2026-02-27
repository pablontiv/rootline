package rules

// ResolveForRecord resolves the effective StemFile for a specific record
// by walking up the directory tree and merging .stem files top-down.
func ResolveForRecord(dir string, recordPath string) (*StemFile, error) {
	entries, err := WalkUp(dir)
	if err != nil {
		return nil, err
	}
	return MergeStemFiles(entries), nil
}
