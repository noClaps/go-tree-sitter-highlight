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
func NewConfiguration(language *tree_sitter.Language, languageName string, highlightsQuery []byte, injectionQuery []byte, localsQuery []byte) (*Configuration, error) {
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
	return &Configuration{
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

type Configuration struct {
	Language                      *tree_sitter.Language
	LanguageName                  string
	Query                         *tree_sitter.Query
	CombinedInjectionsQuery       *tree_sitter.Query
	LocalsPatternIndex            uint
	HighlightsPatternIndex        uint
	HighlightIndices              []*Highlight
	NonLocalVariablePatterns      []bool
	InjectionContentCaptureIndex  *uint
	InjectionLanguageCaptureIndex *uint
	LocalScopeCaptureIndex        *uint
	LocalDefCaptureIndex          *uint
	LocalDefValueCaptureIndex     *uint
	LocalRefCaptureIndex          *uint
}

// Names gets a slice containing all the highlight names used in the configuration.
func (c *Configuration) Names() []string {
	return c.Query.CaptureNames()
}

// Configure sets the list of recognized highlight names.
//
// Tree-sitter syntax-highlighting queries specify highlights in the form of dot-separated
// highlight names like `punctuation.bracket` and `function.method.builtin`. Consumers of
// these queries can choose to recognize highlights with different levels of specificity.
// For example, the string `function.builtin` will match against `function`
// and `function.builtin.constructor`, but will not match `function.method`.
//
// When highlighting, results are returned as `Highlight` values, which contain the index
// of the matched highlight this list of highlight names.
func (c *Configuration) Configure(recognizedNames []string) {
	highlightIndices := make([]*Highlight, len(c.Query.CaptureNames()))
	for i, captureName := range c.Query.CaptureNames() {
		for {
			j := slices.Index(recognizedNames, captureName)
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
	c.HighlightIndices = highlightIndices
}

// NonconformantCaptureNames returns the list of this configuration's capture names that are neither present in the
// list of predefined 'canonical' names nor start with an underscore (denoting 'private'
// captures used as part of capture internals).
func (c *Configuration) NonconformantCaptureNames(captureNames []string) []string {
	if len(captureNames) == 0 {
		captureNames = StandardCaptureNames
	}

	var nonconformantNames []string
	for _, name := range c.Names() {
		if !(strings.HasPrefix(name, "_") || slices.Contains(captureNames, name)) {
			nonconformantNames = append(nonconformantNames, name)
		}
	}

	return nonconformantNames
}
