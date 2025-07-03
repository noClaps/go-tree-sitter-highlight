package highlight

import (
	"fmt"
	"slices"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

const (
	captureInjectionCombined        = "injection.combined"
	captureInjectionLanguage        = "injection.language"
	captureInjectionSelf            = "injection.self"
	captureInjectionParent          = "injection.parent"
	captureInjectionIncludeChildren = "injection.include-children"
	captureLocal                    = "local"
	captureLocalScopeInherits       = "local.scope-inherits"
)

// NewConfiguration creates a new highlight configuration from a [tree_sitter.Language] and a set of queries.
func NewConfiguration(language *tree_sitter.Language, languageName string, highlightsQuery []byte, injectionQuery []byte, localsQuery []byte, recognisedNames []string) (*Configuration, error) {
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
			return setting.Key == captureInjectionCombined
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
			return !predicate.Positive && predicate.Property.Key == captureLocal
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

	highlightIndices := make([]*Highlight, len(query.CaptureNames()))
	for i, captureName := range query.CaptureNames() {
		for {
			j := slices.Index(recognisedNames, captureName)
			if j != -1 {
				index := Highlight(j)
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

	return &Configuration{
		language:                      language,
		languageName:                  languageName,
		query:                         query,
		combinedInjectionsQuery:       combinedInjectionsQuery,
		localsPatternIndex:            localsPatternIndex,
		highlightsPatternIndex:        highlightsPatternIndex,
		highlightIndices:              highlightIndices,
		nonLocalVariablePatterns:      nonLocalVariablePatterns,
		injectionContentCaptureIndex:  injectionContentCaptureIndex,
		injectionLanguageCaptureIndex: injectionLanguageCaptureIndex,
		localScopeCaptureIndex:        localScopeCaptureIndex,
		localDefCaptureIndex:          localDefCaptureIndex,
		localDefValueCaptureIndex:     localDefValueCaptureIndex,
		localRefCaptureIndex:          localRefCaptureIndex,
	}, nil
}

type Configuration struct {
	language                      *tree_sitter.Language
	languageName                  string
	query                         *tree_sitter.Query
	combinedInjectionsQuery       *tree_sitter.Query
	localsPatternIndex            uint
	highlightsPatternIndex        uint
	highlightIndices              []*Highlight
	nonLocalVariablePatterns      []bool
	injectionContentCaptureIndex  *uint
	injectionLanguageCaptureIndex *uint
	localScopeCaptureIndex        *uint
	localDefCaptureIndex          *uint
	localDefValueCaptureIndex     *uint
	localRefCaptureIndex          *uint
}
