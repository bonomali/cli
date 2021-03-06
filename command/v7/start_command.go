package v7

import (
	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/flag"
	v6shared "code.cloudfoundry.org/cli/command/v6/shared"
	"code.cloudfoundry.org/cli/command/v7/shared"
)

//go:generate counterfeiter . StartActor

type StartActor interface {
	GetApplicationByNameAndSpace(appName string, spaceGUID string) (v7action.Application, v7action.Warnings, error)
	GetApplicationSummaryByNameAndSpace(appName string, spaceGUID string, withObfuscatedValues bool, routeActor v7action.RouteActor) (v7action.ApplicationSummary, v7action.Warnings, error)
	PollStart(appGUID string, noWait bool) (v7action.Warnings, error)
	StartApplication(appGUID string) (v7action.Warnings, error)
}

type StartCommand struct {
	RequiredArgs        flag.AppName `positional-args:"yes"`
	usage               interface{}  `usage:"CF_NAME start APP_NAME"`
	relatedCommands     interface{}  `related_commands:"apps, logs, scale, ssh, stop, restart, run-task"`
	envCFStagingTimeout interface{}  `environmentName:"CF_STAGING_TIMEOUT" environmentDescription:"Max wait time for buildpack staging, in minutes" environmentDefault:"15"`
	envCFStartupTimeout interface{}  `environmentName:"CF_STARTUP_TIMEOUT" environmentDescription:"Max wait time for app instance startup, in minutes" environmentDefault:"5"`

	UI          command.UI
	Config      command.Config
	SharedActor command.SharedActor
	Actor       StartActor
	RouteActor  v7action.RouteActor
}

func (cmd *StartCommand) Setup(config command.Config, ui command.UI) error {
	cmd.UI = ui
	cmd.Config = config
	cmd.SharedActor = sharedaction.NewActor(config)

	ccClient, _, err := shared.NewClients(config, ui, true, "")
	if err != nil {
		return err
	}

	ccClientV2, uaaClientV2, err := v6shared.NewClients(config, ui, true)
	if err != nil {
		return err
	}

	cmd.Actor = v7action.NewActor(ccClient, config, nil, nil)
	v2Actor := v2action.NewActor(ccClientV2, uaaClientV2, config)
	cmd.RouteActor = v2Actor

	return nil
}

func (cmd StartCommand) Execute(args []string) error {
	err := cmd.SharedActor.CheckTarget(true, true)
	if err != nil {
		return err
	}

	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	app, warnings, err := cmd.Actor.GetApplicationByNameAndSpace(cmd.RequiredArgs.AppName, cmd.Config.TargetedSpace().GUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}

	if app.Started() {
		cmd.UI.DisplayText("App '{{.AppName}}' is already started.",
			map[string]interface{}{
				"AppName": cmd.RequiredArgs.AppName,
			})
		cmd.UI.DisplayOK()
		return nil
	}

	cmd.UI.DisplayTextWithFlavor("Starting app {{.AppName}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
		"AppName":   cmd.RequiredArgs.AppName,
		"OrgName":   cmd.Config.TargetedOrganization().Name,
		"SpaceName": cmd.Config.TargetedSpace().Name,
		"Username":  user.Name,
	})

	warnings, err = cmd.Actor.StartApplication(app.GUID)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}
	cmd.UI.DisplayText("\nWaiting for app to start...")

	warnings, err = cmd.Actor.PollStart(app.GUID, false)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}

	appSummaryDisplayer := shared.NewAppSummaryDisplayer(cmd.UI)
	summary, warnings, err := cmd.Actor.GetApplicationSummaryByNameAndSpace(
		cmd.RequiredArgs.AppName,
		cmd.Config.TargetedSpace().GUID,
		false,
		cmd.RouteActor,
	)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}

	appSummaryDisplayer.AppDisplay(summary, false)
	return nil
}
