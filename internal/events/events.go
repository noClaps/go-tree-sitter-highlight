package events

import "github.com/noclaps/go-tree-sitter-highlight/types"

// Event is an interface that represents a highlight Event.
// Possible implementations are:
// - [EventLayerStart]
// - [EventLayerEnd]
// - [EventCaptureStart]
// - [EventCaptureEnd]
// - [EventSource]
type Event interface {
	highlightEvent()
}

// EventSource is emitted when a source code range is highlighted.
type EventSource struct {
	StartByte uint
	EndByte   uint
}

func (EventSource) highlightEvent() {}

// EventLayerStart is emitted when a language injection starts.
type EventLayerStart struct {
	// LanguageName is the name of the language that is being injected.
	LanguageName string
}

func (EventLayerStart) highlightEvent() {}

// EventLayerEnd is emitted when a language injection ends.
type EventLayerEnd struct{}

func (EventLayerEnd) highlightEvent() {}

// EventCaptureStart is emitted when a highlight region starts.
type EventCaptureStart struct {
	// Highlight is the capture name of the highlight.
	Highlight types.CaptureIndex
}

func (EventCaptureStart) highlightEvent() {}

// EventCaptureEnd is emitted when a highlight region ends.
type EventCaptureEnd struct{}

func (EventCaptureEnd) highlightEvent() {}
