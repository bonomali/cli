package v7_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/command/commandfakes"
	. "code.cloudfoundry.org/cli/command/v7"
	"code.cloudfoundry.org/cli/command/v7/v7fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("routes Command", func() {
	var (
		cmd             RoutesCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v7fakes.FakeRoutesActor
		executeErr      error
		orglevel        bool
		args            []string
		binaryName      string
	)

	const tableHeaders = `space\s+host\s+domain\s+path\s+apps`

	BeforeEach(func() {
		testUI = ui.NewTestUI(nil, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v7fakes.FakeRoutesActor)
		args = nil

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
	})

	JustBeforeEach(func() {
		cmd = RoutesCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
			Orglevel:    orglevel,
		}
		executeErr = cmd.Execute(args)
	})

	When("the environment is not setup correctly", func() {
		When("checking target fails", func() {
			BeforeEach(func() {
				fakeSharedActor.CheckTargetReturns(actionerror.NotLoggedInError{BinaryName: binaryName})
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: binaryName}))

				Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
				checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
				Expect(checkTargetedOrg).To(BeTrue())
				Expect(checkTargetedSpace).To(BeTrue())
			})
		})

		When("when there is no org targeted", func() {
			BeforeEach(func() {
				fakeSharedActor.CheckTargetReturns(actionerror.NoOrganizationTargetedError{BinaryName: binaryName})
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(actionerror.NoOrganizationTargetedError{BinaryName: binaryName}))
				checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
				Expect(checkTargetedOrg).To(BeTrue())
				Expect(checkTargetedSpace).To(BeTrue())
			})
		})
	})

	Context("When the environment is setup correctly", func() {
		BeforeEach(func() {
			fakeConfig.CurrentUserReturns(configv3.User{Name: "banana"}, nil)
		})

		When("Orglevel is not passed", func() {
			BeforeEach(func() {
				orglevel = false
			})
			When("GetRoutesBySpace returns an error", func() {
				var expectedErr error

				BeforeEach(func() {
					warnings := v7action.Warnings{"warning-1", "warning-2"}
					expectedErr = errors.New("some-error")
					fakeActor.GetRoutesBySpaceReturns(nil, warnings, expectedErr)
				})

				It("prints that error with warnings", func() {
					Expect(executeErr).To(Equal(expectedErr))

					Expect(testUI.Err).To(Say("warning-1"))
					Expect(testUI.Err).To(Say("warning-2"))
					Expect(testUI.Out).ToNot(Say(tableHeaders))
				})
			})

			When("GetRoutesBySpace returns some routes", func() {
				var (
					routes       []v7action.Route
					apps         []v7action.Application
					destinations []v7action.RouteDestination
				)

				BeforeEach(func() {
					routes = []v7action.Route{
						{DomainName: "domain3", GUID: "route-guid-3", SpaceName: "space-3", Host: "host-1"},
						{DomainName: "domain1", GUID: "route-guid-1", SpaceName: "space-1"},
						{DomainName: "domain2", GUID: "route-guid-2", SpaceName: "space-2", Host: "host-3", Path: "/path/2"},
					}
					apps = []v7action.Application{
						{Name: "app1", GUID: "app-guid-1"},
						{Name: "app2", GUID: "app-guid-2"},
					}
					destinations = []v7action.RouteDestination{
						{GUID: "dst-guid-1", App: v7action.RouteDestinationApp{GUID: "app-guid-1"}},
						{GUID: "dst-guid-2", App: v7action.RouteDestinationApp{GUID: "app-guid-2"}},
					}

					fakeActor.GetRoutesBySpaceReturns(
						routes,
						v7action.Warnings{"actor-warning-1"},
						nil,
					)

					fakeActor.GetApplicationsByGUIDsReturns(
						apps,
						v7action.Warnings{"actor-warning-2", "actor-warning-3"},
						nil,
					)

					fakeActor.GetRouteDestinationsReturns(
						[]v7action.RouteDestination{},
						v7action.Warnings{"actor-warning-4"},
						nil,
					)

					fakeActor.GetRouteDestinationsReturnsOnCall(0,
						destinations,
						v7action.Warnings{"actor-warning-4"},
						nil,
					)

					fakeConfig.TargetedOrganizationReturns(configv3.Organization{
						GUID: "some-org-guid",
						Name: "some-org",
					})

					fakeConfig.TargetedSpaceReturns(configv3.Space{
						GUID: "some-space-guid",
						Name: "some-space",
					})
				})

				It("asks the RoutesActor for a list of routes", func() {
					Expect(fakeActor.GetRoutesBySpaceCallCount()).To(Equal(1))
				})

				It("prints warnings", func() {
					Expect(testUI.Err).To(Say("actor-warning-1"))
					Expect(testUI.Err).To(Say("actor-warning-2"))
					Expect(testUI.Err).To(Say("actor-warning-3"))
				})

				It("prints the list of routes in alphabetical order", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(testUI.Out).To(Say(tableHeaders))
					Expect(testUI.Out).To(Say(`space-1\s+domain1\s+`))
					Expect(testUI.Out).To(Say(`space-2\s+host-3\s+domain2\s+\/path\/2`))
					Expect(testUI.Out).To(Say(`space-3\s+host-1\s+domain3\s+app1, app2`))
				})

				It("prints the flavor text", func() {
					Expect(testUI.Out).To(Say("Getting routes for org some-org / space some-space as banana...\n\n"))
				})
			})

			When("GetRoutesBySpace returns no routes", func() {
				var routes []v7action.Route

				BeforeEach(func() {
					routes = []v7action.Route{}

					fakeActor.GetRoutesBySpaceReturns(
						routes,
						v7action.Warnings{"actor-warning-1", "actor-warning-2", "actor-warning-3"},
						nil,
					)

					fakeConfig.TargetedOrganizationReturns(configv3.Organization{
						GUID: "some-org-guid",
						Name: "some-org",
					})
					fakeConfig.TargetedSpaceReturns(configv3.Space{
						GUID: "some-space-guid",
						Name: "some-space",
					})
				})

				It("asks the RoutesActor for a list of routes", func() {
					Expect(fakeActor.GetRoutesBySpaceCallCount()).To(Equal(1))
				})

				It("prints warnings", func() {
					Expect(testUI.Err).To(Say("actor-warning-1"))
					Expect(testUI.Err).To(Say("actor-warning-2"))
					Expect(testUI.Err).To(Say("actor-warning-3"))
				})

				It("does not print table headers", func() {
					Expect(testUI.Out).NotTo(Say(tableHeaders))
				})

				It("prints a message indicating that no routes were found", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(testUI.Out).To(Say("No routes found."))
				})

				It("prints the flavor text", func() {
					Expect(testUI.Out).To(Say("Getting routes for org some-org / space some-space as banana...\n\n"))
				})
			})
		})

		When("Orglevel is passed", func() {
			BeforeEach(func() {
				orglevel = true
			})

			When("GetRoutesByOrg returns an error", func() {
				var expectedErr error

				BeforeEach(func() {
					warnings := v7action.Warnings{"warning-1", "warning-2"}
					expectedErr = errors.New("some-error")
					fakeActor.GetRoutesByOrgReturns(nil, warnings, expectedErr)
				})

				It("prints that error with warnings", func() {
					Expect(executeErr).To(Equal(expectedErr))
					Expect(testUI.Err).To(Say("warning-1"))
					Expect(testUI.Err).To(Say("warning-2"))
					Expect(testUI.Out).ToNot(Say(tableHeaders))
				})
			})

			When("GetRoutesByOrg returns some routes", func() {
				var routes []v7action.Route

				BeforeEach(func() {
					routes = []v7action.Route{
						{DomainName: "domain1", GUID: "route-guid-1", SpaceName: "space-1"},
						{DomainName: "domain2", GUID: "route-guid-2", SpaceName: "space-2", Host: "host-2", Path: "/path/2"},
						{DomainName: "domain3", GUID: "route-guid-3", SpaceName: "space-3", Host: "host-3"},
					}

					fakeActor.GetRoutesByOrgReturns(
						routes,
						v7action.Warnings{"actor-warning-1", "actor-warning-2", "actor-warning-3"},
						nil,
					)

					fakeConfig.TargetedOrganizationReturns(configv3.Organization{
						GUID: "some-org-guid",
						Name: "some-org",
					})
				})

				It("asks the RoutesActor for a list of routes", func() {
					Expect(fakeActor.GetRoutesByOrgCallCount()).To(Equal(1))
				})

				It("prints warnings", func() {
					Expect(testUI.Err).To(Say("actor-warning-1"))
					Expect(testUI.Err).To(Say("actor-warning-2"))
					Expect(testUI.Err).To(Say("actor-warning-3"))
				})

				It("prints the list of routes", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(testUI.Out).To(Say(tableHeaders))
					Expect(testUI.Out).To(Say(`space-1\s+domain1`))
					Expect(testUI.Out).To(Say(`space-2\s+host-2\s+domain2\s+\/path\/2`))
					Expect(testUI.Out).To(Say(`space-3\s+host-3\s+domain3`))
				})

				It("prints the flavor text", func() {
					Expect(testUI.Out).To(Say("Getting routes for org some-org as banana...\n\n"))
				})
			})

			When("GetRoutesByOrg returns no routes", func() {
				var routes []v7action.Route

				BeforeEach(func() {
					routes = []v7action.Route{}

					fakeActor.GetRoutesByOrgReturns(
						routes,
						v7action.Warnings{"actor-warning-1", "actor-warning-2", "actor-warning-3"},
						nil,
					)

					fakeConfig.TargetedOrganizationReturns(configv3.Organization{
						GUID: "some-org-guid",
						Name: "some-org",
					})
				})

				It("asks the RoutesActor for a list of routes", func() {
					Expect(fakeActor.GetRoutesByOrgCallCount()).To(Equal(1))
				})

				It("prints warnings", func() {
					Expect(testUI.Err).To(Say("actor-warning-1"))
					Expect(testUI.Err).To(Say("actor-warning-2"))
					Expect(testUI.Err).To(Say("actor-warning-3"))
				})

				It("does not print table headers", func() {
					Expect(testUI.Out).NotTo(Say(tableHeaders))
				})

				It("prints a message indicating that no routes were found", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(testUI.Out).To(Say("No routes found."))
				})

				It("prints the flavor text", func() {
					Expect(testUI.Out).To(Say("Getting routes for org some-org as banana...\n\n"))
				})
			})
		})
	})
})
