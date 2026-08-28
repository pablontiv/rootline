package skilldist

import (
	"os"
	"path/filepath"
)

func SupportedDestinations(home string) []Destination {
	return []Destination{
		{ID: DestinationClaude, Path: filepath.Join(home, ".claude", "skills", "rootline")},
		{ID: DestinationAgents, Path: filepath.Join(home, ".agents", "skills", "rootline")},
	}
}

func InventoryDestinations(home string, source Source) ([]DestinationState, error) {
	destinations := SupportedDestinations(home)
	states := make([]DestinationState, 0, len(destinations))
	sourceCanonical, err := filepath.EvalSymlinks(source.SkillPath)
	if err != nil {
		return nil, err
	}

	for _, destination := range destinations {
		state, err := inventoryDestination(destination, source, sourceCanonical)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func inventoryDestination(destination Destination, source Source, sourceCanonical string) (DestinationState, error) {
	state := DestinationState{ID: destination.ID, Path: destination.Path}
	info, err := os.Lstat(destination.Path)
	if err != nil {
		if os.IsNotExist(err) {
			state.Kind = KindAbsent
			return state, nil
		}
		return DestinationState{}, err
	}

	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return inventorySymlinkDestination(state, source, sourceCanonical)
	}
	if info.IsDir() {
		digest, err := DigestTree(destination.Path)
		if err != nil {
			return DestinationState{}, err
		}
		state.Kind = KindDirectory
		state.Digest = digest
		return state, nil
	}

	state.Kind = KindUnsupported
	return state, nil
}

func inventorySymlinkDestination(state DestinationState, source Source, sourceCanonical string) (DestinationState, error) {
	lexicalTarget, err := os.Readlink(state.Path)
	if err != nil {
		return DestinationState{}, err
	}
	state.LexicalTarget = lexicalTarget
	state.Kind = KindDivergentSymlink

	canonicalTarget, err := filepath.EvalSymlinks(state.Path)
	if err != nil {
		return state, nil
	}
	state.CanonicalTarget = canonicalTarget

	digest, err := DigestTree(canonicalTarget)
	if err != nil {
		return state, nil
	}
	state.Digest = digest

	if filepath.Clean(lexicalTarget) == filepath.Clean(source.SkillPath) &&
		filepath.Clean(canonicalTarget) == filepath.Clean(sourceCanonical) &&
		digest == source.Digest {
		state.Kind = KindCorrectSymlink
	}
	return state, nil
}
