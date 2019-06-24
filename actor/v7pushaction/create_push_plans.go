package v7pushaction

import (
	"code.cloudfoundry.org/cli/util/manifestparser"
)

// CreatePushPlans returns a set of PushPlan objects based off the inputs
// provided. It's assumed that all flag and argument and manifest combinations
// have been validated prior to calling this function.
func (actor Actor) CreatePushPlans(appNameArg string, spaceGUID string, orgGUID string, parser ManifestParser, overrides FlagOverrides) ([]PushPlan, error) {
	var pushPlans []PushPlan

	eligibleApps, err := getEligibleApplications(parser, appNameArg)
	if err != nil {
		return nil, err
	}

	for _, manifestApplication := range eligibleApps {
		plan := PushPlan{
			OrgGUID:   orgGUID,
			SpaceGUID: spaceGUID,
		}

		// List of PreparePushPlanSequence is defined in NewActor
		for _, updatePlan := range actor.PreparePushPlanSequence {
			var err error
			plan, err = updatePlan(plan, overrides, manifestApplication)
			if err != nil {
				return nil, err
			}
		}

		pushPlans = append(pushPlans, plan)
	}

	return pushPlans, nil
}

func getEligibleApplications(parser ManifestParser, appNameArg string) ([]manifestparser.Application, error) {
	if appNameArg == "" {
		return parser.Apps("")
	}

	if parser.ContainsMultipleApps() {
		return parser.Apps(appNameArg)
	}

	app, err := getApplicationWithName(parser, appNameArg)
	if err != nil {
		return nil, err
	}
	return []manifestparser.Application{app}, nil
}

func getApplicationWithName(parser ManifestParser, appNameArg string) (manifestparser.Application, error) {
	manifestApp := manifestparser.Application{}
	if parser.ContainsManifest() {
		manifestApps, err := parser.Apps("")
		if err != nil {
			return manifestparser.Application{}, err
		}
		manifestApp = manifestApps[0]
	}

	manifestApp.Name = appNameArg
	return manifestApp, nil
}
