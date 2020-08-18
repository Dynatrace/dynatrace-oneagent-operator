package version

type VersionMatcher interface {
	IsLatest() (bool, error)
}
