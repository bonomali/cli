package v7pushaction

type PushEvent struct {
	Event    Event
	Plan     PushPlan
	Err      error
	Warnings Warnings
}

type Event string

const (
	ApplicationAlreadyExists        Event = "App already exists"
	ApplyManifest                   Event = "Applying manifest"
	ApplyManifestComplete           Event = "Applying manifest Complete"
	BoundRoutes                     Event = "bound routes"
	BoundServices                   Event = "bound services"
	ConfiguringServices             Event = "configuring services"
	CreatedApplication              Event = "created application"
	CreatedRoutes                   Event = "created routes"
	CreatingAndMappingRoutes        Event = "creating and mapping routes"
	CreatingApplication             Event = "creating application"
	CreatingArchive                 Event = "creating archive"
	CreatingDroplet                 Event = "creating droplet"
	CreatingPackage                 Event = "creating package"
	PollingBuild                    Event = "polling build"
	ReadingArchive                  Event = "reading archive"
	ResourceMatching                Event = "resource matching"
	RestartingApplication           Event = "restarting application"
	RestartingApplicationComplete   Event = "restarting application complete"
	RetryUpload                     Event = "retry upload"
	ScaleWebProcess                 Event = "scaling the web process"
	ScaleWebProcessComplete         Event = "scaling the web process complete"
	SetDockerImage                  Event = "setting docker properties"
	SetDockerImageComplete          Event = "completed setting docker properties"
	SetDropletComplete              Event = "set droplet complete"
	SetProcessConfiguration         Event = "setting configuration on the process"
	SetProcessConfigurationComplete Event = "completed setting configuration on the process"
	SettingDroplet                  Event = "setting droplet"
	SettingUpApplication            Event = "setting up application"
	SkippingApplicationCreation     Event = "skipping creation"
	StagingComplete                 Event = "staging complete"
	StartingDeployment              Event = "starting deployment"
	StartingStaging                 Event = "starting staging"
	StoppingApplication             Event = "stopping application"
	StoppingApplicationComplete     Event = "stopping application complete"
	UnmappingRoutes                 Event = "unmapping routes"
	UpdatedApplication              Event = "updated application"
	UploadDropletComplete           Event = "upload droplet complete"
	UploadingApplication            Event = "uploading application"
	UploadingApplicationWithArchive Event = "uploading application with archive"
	UploadingDroplet                Event = "uploading droplet"
	UploadWithArchiveComplete       Event = "upload complete"
	WaitingForDeployment            Event = "waiting for deployment"
	Complete                        Event = "complete"
)
