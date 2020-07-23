package istio

import (
	"fmt"
	"meshery-operator/pkg/meshsync/models"

	"istio.io/pkg/version"
)

// event represents an istio event
type event struct {
	evType models.EventType
	info   *version.MeshInfo
}

// Details implements the `Details()` method of the Event interface
func (ev *event) Details() models.MeshInfo {
	v := ev.String()
	info := models.MeshInfo{
		Version: &v,
	}

	return info
}

// String satifies the Stringer interface
func (ev *event) String() string {
	var v string
	for _, comp := range *ev.info {
		v = v + fmt.Sprintf("[component] %s [version] %s, ", comp.Component, comp.Info.LongForm())
	}
	return v
}

// Type returns the event type being processed
func (ev *event) Type() models.EventType {
	return ev.evType
}
