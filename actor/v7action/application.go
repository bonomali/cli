package v7action

import (
	"errors"
	"time"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccerror"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
)

// Application represents a V3 actor application.
type Application struct {
	Name                string
	GUID                string
	StackName           string
	State               constant.ApplicationState
	LifecycleType       constant.AppLifecycleType
	LifecycleBuildpacks []string
	Metadata            *Metadata
}

func (app Application) Started() bool {
	return app.State == constant.ApplicationStarted
}

func (app Application) Stopped() bool {
	return app.State == constant.ApplicationStopped
}

func (actor Actor) DeleteApplicationByNameAndSpace(name string, spaceGUID string) (Warnings, error) {
	var allWarnings Warnings

	app, getAppWarnings, err := actor.GetApplicationByNameAndSpace(name, spaceGUID)
	allWarnings = append(allWarnings, getAppWarnings...)
	if err != nil {
		return allWarnings, err
	}

	jobURL, deleteAppWarnings, err := actor.CloudControllerClient.DeleteApplication(app.GUID)
	allWarnings = append(allWarnings, deleteAppWarnings...)
	if err != nil {
		return allWarnings, err
	}

	pollWarnings, err := actor.CloudControllerClient.PollJob(jobURL)
	allWarnings = append(allWarnings, pollWarnings...)
	return allWarnings, err
}

func (actor Actor) GetApplicationsByGUIDs(appGUIDs []string) ([]Application, Warnings, error) {
	apps, warnings, err := actor.CloudControllerClient.GetApplications(
		ccv3.Query{Key: ccv3.GUIDFilter, Values: appGUIDs},
	)

	if err != nil {
		return nil, Warnings(warnings), err
	}

	if len(apps) < len(appGUIDs) {
		return nil, Warnings(warnings), actionerror.ApplicationsNotFoundError{}
	}

	actorApps := []Application{}
	for _, a := range apps {
		actorApps = append(actorApps, actor.convertCCToActorApplication(a))
	}

	return actorApps, Warnings(warnings), nil
}

func (actor Actor) GetApplicationsByNamesAndSpace(appNames []string, spaceGUID string) ([]Application, Warnings, error) {
	apps, warnings, err := actor.CloudControllerClient.GetApplications(
		ccv3.Query{Key: ccv3.NameFilter, Values: appNames},
		ccv3.Query{Key: ccv3.SpaceGUIDFilter, Values: []string{spaceGUID}},
	)

	if err != nil {
		return nil, Warnings(warnings), err
	}

	if len(apps) < len(appNames) {
		return nil, Warnings(warnings), actionerror.ApplicationsNotFoundError{}
	}

	actorApps := []Application{}
	for _, a := range apps {
		actorApps = append(actorApps, actor.convertCCToActorApplication(a))
	}
	return actorApps, Warnings(warnings), nil
}

// GetApplicationByNameAndSpace returns the application with the given
// name in the given space.
func (actor Actor) GetApplicationByNameAndSpace(appName string, spaceGUID string) (Application, Warnings, error) {
	apps, warnings, err := actor.GetApplicationsByNamesAndSpace([]string{appName}, spaceGUID)

	if err != nil {
		if _, ok := err.(actionerror.ApplicationsNotFoundError); ok {
			return Application{}, warnings, actionerror.ApplicationNotFoundError{Name: appName}
		}
		return Application{}, warnings, err
	}

	return apps[0], warnings, nil
}

// GetApplicationsBySpace returns all applications in a space.
func (actor Actor) GetApplicationsBySpace(spaceGUID string) ([]Application, Warnings, error) {
	ccApps, warnings, err := actor.CloudControllerClient.GetApplications(
		ccv3.Query{Key: ccv3.SpaceGUIDFilter, Values: []string{spaceGUID}},
	)

	if err != nil {
		return []Application{}, Warnings(warnings), err
	}

	var apps []Application
	for _, ccApp := range ccApps {
		apps = append(apps, actor.convertCCToActorApplication(ccApp))
	}
	return apps, Warnings(warnings), nil
}

// CreateApplicationInSpace creates and returns the application with the given
// name in the given space.
func (actor Actor) CreateApplicationInSpace(app Application, spaceGUID string) (Application, Warnings, error) {
	createdApp, warnings, err := actor.CloudControllerClient.CreateApplication(
		ccv3.Application{
			LifecycleType:       app.LifecycleType,
			LifecycleBuildpacks: app.LifecycleBuildpacks,
			StackName:           app.StackName,
			Name:                app.Name,
			Relationships: ccv3.Relationships{
				constant.RelationshipTypeSpace: ccv3.Relationship{GUID: spaceGUID},
			},
		})

	if err != nil {
		if _, ok := err.(ccerror.NameNotUniqueInSpaceError); ok {
			return Application{}, Warnings(warnings), actionerror.ApplicationAlreadyExistsError{Name: app.Name}
		}
		return Application{}, Warnings(warnings), err
	}

	return actor.convertCCToActorApplication(createdApp), Warnings(warnings), nil
}

// SetApplicationProcessHealthCheckTypeByNameAndSpace sets the health check
// information of the provided processType for an application with the given
// name and space GUID.
func (actor Actor) SetApplicationProcessHealthCheckTypeByNameAndSpace(
	appName string,
	spaceGUID string,
	healthCheckType constant.HealthCheckType,
	httpEndpoint string,
	processType string,
	invocationTimeout int64,
) (Application, Warnings, error) {

	app, getWarnings, err := actor.GetApplicationByNameAndSpace(appName, spaceGUID)
	if err != nil {
		return Application{}, getWarnings, err
	}

	setWarnings, err := actor.UpdateProcessByTypeAndApplication(
		processType,
		app.GUID,
		Process{
			HealthCheckType:              healthCheckType,
			HealthCheckEndpoint:          httpEndpoint,
			HealthCheckInvocationTimeout: invocationTimeout,
		})
	return app, append(getWarnings, setWarnings...), err
}

// StopApplication stops an application.
func (actor Actor) StopApplication(appGUID string) (Warnings, error) {
	_, warnings, err := actor.CloudControllerClient.UpdateApplicationStop(appGUID)

	return Warnings(warnings), err
}

// StartApplication starts an application.
func (actor Actor) StartApplication(appGUID string) (Warnings, error) {
	_, warnings, err := actor.CloudControllerClient.UpdateApplicationStart(appGUID)
	return Warnings(warnings), err
}

// RestartApplication restarts an application and waits for it to start.
func (actor Actor) RestartApplication(appGUID string) (Warnings, error) {
	var allWarnings Warnings
	_, warnings, err := actor.CloudControllerClient.UpdateApplicationRestart(appGUID)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return allWarnings, err
	}

	pollingWarnings, err := actor.PollStart(appGUID)
	allWarnings = append(allWarnings, pollingWarnings...)
	return allWarnings, err
}

func (actor Actor) PollStart(appGUID string) (Warnings, error) {
	processes, warnings, err := actor.CloudControllerClient.GetApplicationProcesses(appGUID)
	if err != nil {
		return Warnings(warnings), err
	}
	return actor.pollForProcesses(processes)
}

// pollStartForRolling(appGUID, depGUID, noWait)
// deployment = getdep
// if noWait
// 	processes = dep.NewProcesses
// else
// 	processes = removeWebProcess(getAppProcesses)
// initDeploymentState = deployment.State
// for timeoute
// 	if deploymentState != DEPLOYED || !noWait {
// deploymentState = getDeploymentState()

// switch deploymentState {
// return allWarnings, nil
// case constant.DeploymentCanceled:
// return allWarnings, errors.New("Deployment has been canceled")
// case constant.DeploymentFailed:
// return allWarnings, errors.New("Deployment has failed")
// case constant.DeploymentDeploying:
// case constant.DeploymentFailing:
// case constant.DeploymentCanceling:
// time.Sleep(actor.Config.PollingInterval())
// continue
// }
// }

// if pollProcesses(process) {
// 	break
// }

func (actor Actor) refreshDeployment(staleDeployment ccv3.Deployment) (ccv3.Deployment, Warnings, error) {
	if staleDeployment.State == constant.DeploymentDeployed {
		return staleDeployment, nil, nil
	}

	deployment, warnings, err := actor.CloudControllerClient.GetDeployment(staleDeployment.GUID)
	if err != nil {
		return ccv3.Deployment{}, Warnings(warnings), err
	}

	switch {
	case deployment.State == constant.DeploymentCanceled:
		return ccv3.Deployment{}, Warnings(warnings), errors.New("Deployment has been canceled")
	case deployment.State == constant.DeploymentFailed:
		return ccv3.Deployment{}, Warnings(warnings), errors.New("Deployment has failed")
	}

	return deployment, Warnings(warnings), nil

}

func (actor Actor) getProcesses(deployment ccv3.Deployment, appGUID string, noWait bool) ([]ccv3.Process, Warnings, error) {
	if noWait {
		return deployment.NewProcesses, nil, nil
	}
	processes, warnings, err := actor.CloudControllerClient.GetApplicationProcesses(appGUID)
	return processes, Warnings(warnings), err
}

/*
func (actor Actor) PollStartForRollingEOD(appGUID string, deploymentGUID string, noWait bool) (Warnings, error) {
	var allWarnings Warnings
	var processes []ccv3.Process
	deployment := ccv3.Deployment{GUID: deploymentGUID}

	timeout := time.Now().Add(actor.Config.StartupTimeout())
	for time.Now().Before(timeout) {
		deployment, warnings, err := actor.refreshDeployment(deployment)
		if err != nil {
			return warnings, err
		}

		if deployment.State == constant.DeploymentDeployed || noWait {

			if processes != nil {
				processes, warnings, err = actor.getProcesses(deployment, appGUID, noWait)
				allWarnings = append(allWarnings, warnings...)
				if err != nil {
					return allWarnings, err
				}
			}

			allProcessesDone, warnings, err := actor.PollProcesses(processes)
			allWarnings = append(allWarnings, warnings...)
			if allProcessesDone || err != nil {
				return allWarnings, err
			}
		}
		time.Sleep(actor.Config.PollingInterval())
	}
	return allWarnings, actionerror.StartupTimeoutError{}
}
*/

//replaced by PollStartForRolling(2Loops), but it might be helpful to keep this for reference until --no-wait is committed.
func (actor Actor) PollStartForRolling1Loop(appGUID string, deploymentGUID string, noWait bool) (Warnings, error) {
	var allWarnings Warnings
	var processes []ccv3.Process
	var deployment *ccv3.Deployment

	// cant do this because processes will have both old and deploying processes so we need to get processes once it is deployed
	if !noWait {
		ccProcesses, warnings, err := actor.CloudControllerClient.GetApplicationProcesses(appGUID)
		if err != nil {
			return Warnings(warnings), err
		}
		processes = ccProcesses
	}

	timeout := time.Now().Add(actor.Config.StartupTimeout())
	for time.Now().Before(timeout) {
		if deployment == nil {
			ccDeployment, warnings, err := actor.checkDeployment(deployment.GUID, noWait)
			allWarnings = append(allWarnings, warnings...)
			if err != nil {
				return allWarnings, err
			}
			if ccDeployment != nil {
				deployment = ccDeployment
				if noWait {
					processes = deployment.NewProcesses
				}
			}
		}

		if processes != nil {
			stopPolling, warnings, err := actor.PollProcesses(processes)
			allWarnings = append(allWarnings, warnings...)
			if stopPolling || err != nil {
				return allWarnings, err
			}
		}
		time.Sleep(actor.Config.PollingInterval())
	}
	return allWarnings, actionerror.StartupTimeoutError{}
}


func (actor Actor) PollStartForRolling(appGUID string, deploymentGUID string, noWait bool) (Warnings, error) {
	var allWarnings Warnings
	var processes []ccv3.Process

	timeout := time.Now().Add(actor.Config.StartupTimeout())

	for time.Now().Before(timeout) {
		// Do we have a healthy deployment?
		ccDeployment, warnings, err := actor.checkDeployment(deploymentGUID, noWait)
		allWarnings = append(allWarnings, warnings...)
		if err != nil {
			return allWarnings, err
		}
		if ccDeployment != nil {
			if noWait {
				processes = ccDeployment.NewProcesses
			} else {
				ccProcesses, warnings, err := actor.CloudControllerClient.GetApplicationProcesses(appGUID)
				if err != nil {
					return Warnings(warnings), err
				}
				processes = ccProcesses
			}
			break
		}
		time.Sleep(actor.Config.PollingInterval())
	}

	checkDeployment := false // no need to check deployment first time through this loop
	for time.Now().Before(timeout) {
		if noWait {
			if checkDeployment {
				// Did the user cancel the deployment, or did it fail?
				_, warnings, err := actor.checkDeployment(deploymentGUID, noWait)
				allWarnings = append(allWarnings, warnings...)
				if err != nil {
					return allWarnings, err
				}
			} else {
				checkDeployment = true
			}
		}

		stopPolling, warnings, err := actor.PollProcesses(processes)
		allWarnings = append(allWarnings, warnings...)
		if stopPolling || err != nil {
			return allWarnings, err
		}

		time.Sleep(actor.Config.PollingInterval())


	}
	return allWarnings, actionerror.StartupTimeoutError{}
}

func (actor Actor) checkDeployment(deploymentGUID string, noWait bool) (*ccv3.Deployment, Warnings, error) {
	var allWarnings Warnings
	deployment, warnings, err := actor.CloudControllerClient.GetDeployment(deploymentGUID)
	allWarnings = append(allWarnings, warnings...)
	if err != nil {
		return nil, allWarnings, err
	}

	switch {
	case deployment.State == constant.DeploymentCanceled:
		return nil, allWarnings, errors.New("Deployment has been canceled")
	case deployment.State == constant.DeploymentFailed:
		return nil, allWarnings, errors.New("Deployment has failed")
	case noWait || deployment.State == constant.DeploymentDeployed:
		return &deployment, allWarnings, nil
	default:
		return nil, allWarnings, nil
	}
}

 // PollProcesses - return true if there's no need to keep polling
func (actor Actor) PollProcesses(processes []ccv3.Process) (bool, Warnings, error) {
	numProcesses := len(processes)
	numStableProcesses := 0
	var allWarnings Warnings
	for _, process := range processes {
		ccInstances, ccWarnings, err := actor.CloudControllerClient.GetProcessInstances(process.GUID)
		instances := ProcessInstances(ccInstances)
		allWarnings = append(allWarnings, ccWarnings...)
		if err != nil {
			return true, allWarnings, err
		}

		if instances.Empty() || instances.AnyRunning() {
			numStableProcesses += 1
		} else if instances.AllCrashed() {
			return false, allWarnings, actionerror.AllInstancesCrashedError{}
		} else {
			//precondition: !instances.Empty() && no instances are running
			// do not increment numStableProcesses
			return false, allWarnings, nil
		}
	}
	return numStableProcesses == numProcesses, allWarnings, nil
}

// func (actor Actor) PollStartForRolling(appGUID string, deploymentGUID string, noWait bool) (Warnings, error) {
// 	deploymentWarnings, err := actor.pollDeployment(deploymentGUID)
// 	var allWarnings Warnings
// 	allWarnings = append(allWarnings, deploymentWarnings...)
// 	if err != nil {
// 		return allWarnings, err
// 	}

// 	allProcesses, warnings, err := actor.CloudControllerClient.GetApplicationProcesses(appGUID)
// 	allWarnings = append(allWarnings, warnings...)

// 	if err != nil {
// 		return allWarnings, err
// 	}

// 	pollingWarnings, err := actor.pollForProcesses(allProcesses)
// 	allWarnings = append(allWarnings, pollingWarnings...)
// 	return allWarnings, err
// }

func (actor Actor) getDeploymentState(deploymentGUID string) (constant.DeploymentState, Warnings, error) {
	deployment, warnings, err := actor.CloudControllerClient.GetDeployment(deploymentGUID)
	if err != nil {
		return "", Warnings(warnings), err
	}
	return deployment.State, Warnings(warnings), nil
}

func (actor Actor) pollDeployment(deploymentGUID string) (Warnings, error) {
	var allWarnings Warnings

	timeout := time.Now().Add(actor.Config.StartupTimeout())
	for time.Now().Before(timeout) {
		deploymentState, warnings, err := actor.getDeploymentState(deploymentGUID)
		allWarnings = append(allWarnings, warnings...)
		if err != nil {
			return allWarnings, err
		}
		switch deploymentState {
		case constant.DeploymentDeployed:
			return allWarnings, nil
		case constant.DeploymentCanceled:
			return allWarnings, errors.New("Deployment has been canceled")
		case constant.DeploymentFailed:
			return allWarnings, errors.New("Deployment has failed")
		case constant.DeploymentDeploying:
		case constant.DeploymentFailing:
		case constant.DeploymentCanceling:
			time.Sleep(actor.Config.PollingInterval())
		}
	}
	return allWarnings, actionerror.StartupTimeoutError{}
}

// UpdateApplication updates the buildpacks on an application
func (actor Actor) UpdateApplication(app Application) (Application, Warnings, error) {
	ccApp := ccv3.Application{
		GUID:                app.GUID,
		StackName:           app.StackName,
		LifecycleType:       app.LifecycleType,
		LifecycleBuildpacks: app.LifecycleBuildpacks,
		Metadata:            (*ccv3.Metadata)(app.Metadata),
	}

	updatedApp, warnings, err := actor.CloudControllerClient.UpdateApplication(ccApp)
	if err != nil {
		return Application{}, Warnings(warnings), err
	}

	return actor.convertCCToActorApplication(updatedApp), Warnings(warnings), nil
}

func (Actor) convertCCToActorApplication(app ccv3.Application) Application {
	return Application{
		GUID:                app.GUID,
		StackName:           app.StackName,
		LifecycleType:       app.LifecycleType,
		LifecycleBuildpacks: app.LifecycleBuildpacks,
		Name:                app.Name,
		State:               app.State,
		Metadata:            (*Metadata)(app.Metadata),
	}
}

func (actor Actor) pollForProcesses(processes []ccv3.Process) (Warnings, error) {
	var allWarnings Warnings
	timeout := time.Now().Add(actor.Config.StartupTimeout())
	for time.Now().Before(timeout) {
		allProcessesDone := true
		for _, process := range processes {
			shouldContinuePolling, warnings, err := actor.shouldContinuePollingProcessStatus(process)
			allWarnings = append(allWarnings, warnings...)
			if err != nil {
				return allWarnings, err
			}

			if shouldContinuePolling {
				allProcessesDone = false
				break
			}
		}

		if allProcessesDone {
			return allWarnings, nil
		}
		time.Sleep(actor.Config.PollingInterval())
	}

	return allWarnings, actionerror.StartupTimeoutError{}
}

func (actor Actor) shouldContinuePollingProcessStatus(process ccv3.Process) (bool, Warnings, error) {
	ccInstances, ccWarnings, err := actor.CloudControllerClient.GetProcessInstances(process.GUID)
	instances := ProcessInstances(ccInstances)
	warnings := Warnings(ccWarnings)
	if err != nil {
		return true, warnings, err
	}

	if instances.Empty() || instances.AnyRunning() {
		return false, warnings, nil
	} else if instances.AllCrashed() {
		return false, warnings, actionerror.AllInstancesCrashedError{}
	}

	return true, warnings, nil
}
