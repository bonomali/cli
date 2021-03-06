package v7

import (
	"os"

	"code.cloudfoundry.org/cli/actor/sharedaction"
	"code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/command"
	"code.cloudfoundry.org/cli/command/flag"
	"code.cloudfoundry.org/cli/command/translatableerror"
	"code.cloudfoundry.org/cli/command/v7/shared"
	"code.cloudfoundry.org/cli/util/manifestparser"
)

//go:generate counterfeiter . ApplyManifestActor
type ApplyManifestActor interface {
	SetSpaceManifest(spaceGUID string, rawManifest []byte, noRoute bool) (v7action.Warnings, error)
}

type ApplyManifestCommand struct {
	PathToManifest  flag.ManifestPathWithExistenceCheck `short:"f" description:"Path to app manifest"`
	usage           interface{}                         `usage:"CF_NAME apply-manifest -f APP_MANIFEST_PATH"`
	relatedCommands interface{}                         `related_commands:"create-app, create-app-manifest, push"`

	UI              command.UI
	Config          command.Config
	SharedActor     command.SharedActor
	ManifestLocator ManifestLocator
	Actor           ApplyManifestActor
	Parser          ManifestParser
	CWD             string
}

func (cmd *ApplyManifestCommand) Setup(config command.Config, ui command.UI) error {

	cmd.UI = ui
	cmd.Config = config
	cmd.SharedActor = sharedaction.NewActor(config)

	ccClient, _, err := shared.NewClients(config, ui, true, "")
	if err != nil {
		return err
	}
	cmd.Actor = v7action.NewActor(ccClient, config, nil, nil)

	cmd.ManifestLocator = manifestparser.NewLocator()
	cmd.Parser = manifestparser.NewParser()

	currentDir, err := os.Getwd()
	cmd.CWD = currentDir

	return err
}

func (cmd ApplyManifestCommand) Execute(args []string) error {
	err := cmd.SharedActor.CheckTarget(true, true)
	if err != nil {
		return err
	}

	user, err := cmd.Config.CurrentUser()
	if err != nil {
		return err
	}

	var manifestReadPath = string(cmd.PathToManifest)

	if manifestReadPath == "" {
		locatorPath := cmd.CWD
		resolvedPath, exists, err := cmd.ManifestLocator.Path(locatorPath)

		if err != nil {
			return err
		}

		if !exists {
			return translatableerror.ManifestFileNotFoundInDirectoryError{PathToManifest: locatorPath}
		}

		manifestReadPath = resolvedPath
	}

	cmd.UI.DisplayTextWithFlavor("Applying manifest {{.ManifestPath}} in org {{.OrgName}} / space {{.SpaceName}} as {{.Username}}...", map[string]interface{}{
		"ManifestPath": manifestReadPath,
		"OrgName":      cmd.Config.TargetedOrganization().Name,
		"SpaceName":    cmd.Config.TargetedSpace().Name,
		"Username":     user.Name,
	})

	err = cmd.Parser.InterpolateAndParse(manifestReadPath, nil, nil, "")
	if err != nil {
		return err
	}

	warnings, err := cmd.Actor.SetSpaceManifest(cmd.Config.TargetedSpace().GUID, cmd.Parser.FullRawManifest(), false)
	cmd.UI.DisplayWarnings(warnings)
	if err != nil {
		return err
	}

	cmd.UI.DisplayOK()

	return nil
}
