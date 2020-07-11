package controller

type Event struct {
	key       string
	eventType string
	Mesh      Meshes
}

type Meshes struct {
	Istio
}

type Istio struct {
	virtaulService string
}
