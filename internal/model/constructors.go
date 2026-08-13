package model

// NewMib returns an empty, initialized Mib.
func NewMib() *Mib {
	return &Mib{
		root:               &Node{kind: KindInternal},
		moduleByName:       make(map[string]*Module),
		nameToNodes:        make(map[string][]*Node),
		objectByName:       make(map[string]*Object),
		typeByName:         make(map[string]*Type),
		notificationByName: make(map[string]*Notification),
		groupByName:        make(map[string]*Group),
		complianceByName:   make(map[string]*Compliance),
		capabilityByName:   make(map[string]*Capability),
	}
}

// NewModule returns a Module initialized with the given name.
func NewModule(name string) *Module {
	return &Module{
		name:                name,
		objectsByName:       make(map[string]*Object),
		typesByName:         make(map[string]*Type),
		notificationsByName: make(map[string]*Notification),
		groupsByName:        make(map[string]*Group),
		compliancesByName:   make(map[string]*Compliance),
		capabilitiesByName:  make(map[string]*Capability),
		nodesByName:         make(map[string]*Node),
	}
}

// NewObject returns an Object initialized with the given name.
func NewObject(name string) *Object {
	return &Object{entity: entity{name: name}}
}

// NewType returns a Type initialized with the given name.
func NewType(name string) *Type {
	return &Type{name: name}
}

// NewNotification returns a Notification initialized with the given name.
func NewNotification(name string) *Notification {
	return &Notification{entity: entity{name: name}}
}

// NewGroup returns a Group initialized with the given name.
func NewGroup(name string) *Group {
	return &Group{entity: entity{name: name}}
}

// NewCompliance returns a Compliance initialized with the given name.
func NewCompliance(name string) *Compliance {
	return &Compliance{entity: entity{name: name}}
}

// NewCapability returns a Capability initialized with the given name.
func NewCapability(name string) *Capability {
	return &Capability{entity: entity{name: name}}
}

// ClassifyIndexEncoding determines the index encoding from the object's
// resolved type and effective constraints. Returns IndexEncodingUnknown
// for bare type indexes or unresolved types.
func ClassifyIndexEncoding(obj *Object, implied bool) IndexEncoding {
	return classifyIndexEncoding(obj, implied)
}
