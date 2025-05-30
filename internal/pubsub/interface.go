package pubsub

import "github.com/nubogo/nubo/language"

// Event represents a pubsub event.
type Event struct {
	// ID is the unique identifier of the event.
	ID string
	// Args is the list of arguments of the event.
	Args []language.FnArg
}

// TransportData represents the data to be transported.
type TransportData []language.Object

// UnsubscribeFunc represents a function that unsubscribes from a topic.
type UnsubscribeFunc func() error

// Provider represents a pubsub provider.
type Provider interface {
	// Events returns a list of events.
	Events() []Event
	// AddEvent adds a new event to the provider.
	AddEvent(Event)

	// Publish publishes a new event to the provider.
	Publish(topic string, data TransportData) error
	// Subscribe subscribes to a topic.
	Subscribe(topic string, handler func(TransportData)) (UnsubscribeFunc, error)

	// Close closes the provider.
	Close() error
}
