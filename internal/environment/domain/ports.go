package environment

// EnvironmentStore persists environments and the globals scope, stored as
// project-root siblings of the collections/ directory. Implemented by
// environment/infrastructure.
type EnvironmentStore interface {
	ListEnvironments() ([]string, error)
	LoadEnvironment(name string) (Environment, error)
	SaveEnvironment(env Environment) error
	DeleteEnvironment(name string) error
	LoadGlobals() (Environment, error)
	SaveGlobals(env Environment) error
	// ActiveName/SetActiveName persist which environment was last made
	// active, so a restart can restore it instead of resetting to none.
	ActiveName() (string, error)
	SetActiveName(name string) error
}
