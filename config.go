package highlight

import (
	"fmt"
	"slices"
	"strings"

	"github.com/noclaps/go-tree-sitter-highlight/internal/highlight"
	"github.com/noclaps/go-tree-sitter-highlight/types"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// NewConfiguration creates a new highlight configuration from a [tree_sitter.Language] and a set of queries.
func NewConfiguration(lang types.Language, recognisedNames []string) (*types.Configuration, error) {
	injectionQuery := lang.InjectionQuery
	localsQuery := lang.LocalsQuery
	highlightsQuery := lang.HighlightsQuery
	language := lang.Lang
	languageName := lang.Name

	querySource := injectionQuery
	localsQueryOffset := uint(len(querySource))
	querySource = append(querySource, localsQuery...)
	highlightsQueryOffset := uint(len(querySource))
	querySource = append(querySource, highlightsQuery...)

	query, err := tree_sitter.NewQuery(language, string(querySource))
	if err != nil {
		return nil, fmt.Errorf("error creating query: %w", err)
	}

	localsPatternIndex := uint(0)
	highlightsPatternIndex := uint(0)
	for i := range query.PatternCount() {
		patternOffset := query.StartByteForPattern(i)
		if patternOffset < highlightsQueryOffset {
			if patternOffset < highlightsQueryOffset {
				highlightsPatternIndex++
			}
			if patternOffset < localsQueryOffset {
				localsPatternIndex++
			}
		}
	}

	combinedInjectionsQuery, err := tree_sitter.NewQuery(language, string(injectionQuery))
	if err != nil {
		return nil, fmt.Errorf("error creating combined injections query: %w", err)
	}
	var hasCombinedQueries bool
	for i := range localsPatternIndex {
		settings := combinedInjectionsQuery.PropertySettings(i)
		if slices.ContainsFunc(settings, func(setting tree_sitter.QueryProperty) bool {
			return setting.Key == highlight.CaptureInjectionCombined
		}) {
			hasCombinedQueries = true
			query.DisablePattern(i)
		} else {
			combinedInjectionsQuery.DisablePattern(i)
		}
	}
	if !hasCombinedQueries {
		combinedInjectionsQuery = nil
	}

	nonLocalVariablePatterns := make([]bool, 0)
	for i := range query.PatternCount() {
		predicates := query.PropertyPredicates(i)
		if slices.ContainsFunc(predicates, func(predicate tree_sitter.PropertyPredicate) bool {
			return !predicate.Positive && predicate.Property.Key == highlight.CaptureLocal
		}) {
			nonLocalVariablePatterns = append(nonLocalVariablePatterns, true)
		}
	}

	var (
		injectionContentCaptureIndex  *uint
		injectionLanguageCaptureIndex *uint
		localDefCaptureIndex          *uint
		localDefValueCaptureIndex     *uint
		localRefCaptureIndex          *uint
		localScopeCaptureIndex        *uint
	)

	for i, captureName := range query.CaptureNames() {
		ui := uint(i)
		switch captureName {
		case "injection.content":
			injectionContentCaptureIndex = &ui
		case "injection.language":
			injectionLanguageCaptureIndex = &ui
		case "local.definition":
			localDefCaptureIndex = &ui
		case "local.definition-value":
			localDefValueCaptureIndex = &ui
		case "local.reference":
			localRefCaptureIndex = &ui
		case "local.scope":
			localScopeCaptureIndex = &ui
		}
	}

	highlightIndices := make([]*types.CaptureIndex, len(query.CaptureNames()))
	for i, captureName := range query.CaptureNames() {
		for {
			j := slices.Index(recognisedNames, captureName)
			if j != -1 {
				index := types.CaptureIndex(j)
				highlightIndices[i] = &index
				break
			}

			lastDot := strings.LastIndex(captureName, ".")
			if lastDot == -1 {
				break
			}
			captureName = captureName[:lastDot]
		}
	}

	return &types.Configuration{
		Language:                      language,
		LanguageName:                  languageName,
		Query:                         query,
		CombinedInjectionsQuery:       combinedInjectionsQuery,
		LocalsPatternIndex:            localsPatternIndex,
		HighlightsPatternIndex:        highlightsPatternIndex,
		HighlightIndices:              highlightIndices,
		NonLocalVariablePatterns:      nonLocalVariablePatterns,
		InjectionContentCaptureIndex:  injectionContentCaptureIndex,
		InjectionLanguageCaptureIndex: injectionLanguageCaptureIndex,
		LocalScopeCaptureIndex:        localScopeCaptureIndex,
		LocalDefCaptureIndex:          localDefCaptureIndex,
		LocalDefValueCaptureIndex:     localDefValueCaptureIndex,
		LocalRefCaptureIndex:          localRefCaptureIndex,
	}, nil
}
