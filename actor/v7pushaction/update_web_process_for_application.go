package v7pushaction

import (
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3/constant"
	log "github.com/sirupsen/logrus"
)

func (actor Actor) UpdateWebProcessForApplication(pushPlan PushPlan, eventStream chan<- *PushEvent, progressBar ProgressBar) (PushPlan, Warnings, error) {
	log.Info("Setting Web Process's Configuration")
	eventStream <- &PushEvent{Plan: pushPlan, Event: SetProcessConfiguration}

	log.WithField("Process", pushPlan.UpdateWebProcess).Debug("Update process")
	warnings, err := actor.V7Actor.UpdateProcessByTypeAndApplication(constant.ProcessTypeWeb, pushPlan.Application.GUID, pushPlan.UpdateWebProcess)
	if err != nil {
		return pushPlan, Warnings(warnings), err
	}
	eventStream <- &PushEvent{Plan: pushPlan, Event: SetProcessConfigurationComplete}
	return pushPlan, Warnings(warnings), nil
}
