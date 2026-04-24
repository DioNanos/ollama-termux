package launch

import "strings"

// OverrideIntegration replaces one registry entry's runner for tests and returns a restore function.
func OverrideIntegration(name string, runner Runner) func() {
	spec, err := LookupIntegrationSpec(name)
	if err != nil {
		key := strings.ToLower(name)
		newSpec := &IntegrationSpec{Name: key, Runner: runner}
		integrationSpecs = append(integrationSpecs, newSpec)
		rebuildIntegrationSpecIndexes()
		return func() {
			for i, spec := range integrationSpecs {
				if spec.Name == key {
					integrationSpecs = append(integrationSpecs[:i], integrationSpecs[i+1:]...)
					break
				}
			}
			rebuildIntegrationSpecIndexes()
		}
	}

	original := spec.Runner
	spec.Runner = runner
	return func() {
		spec.Runner = original
	}
}
