package v7pushaction_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/v7action"
	. "code.cloudfoundry.org/cli/actor/v7pushaction"
	"code.cloudfoundry.org/cli/actor/v7pushaction/v7pushactionfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateDeploymentForApplication()", func() {
	var (
		actor       *Actor
		fakeV7Actor *v7pushactionfakes.FakeV7Actor

		returnedPushPlan PushPlan
		paramPlan        PushPlan
		fakeProgressBar  *v7pushactionfakes.FakeProgressBar

		warnings   Warnings
		executeErr error

		events []Event
	)

	BeforeEach(func() {
		actor, _, fakeV7Actor, _ = getTestPushActor()

		fakeProgressBar = new(v7pushactionfakes.FakeProgressBar)

		paramPlan = PushPlan{
			Application: v7action.Application{
				GUID: "some-app-guid",
			},
		}
	})

	JustBeforeEach(func() {
		events = EventFollower(func(eventStream chan<- Event) {
			returnedPushPlan, warnings, executeErr = actor.CreateDeploymentForApplication(paramPlan, eventStream, fakeProgressBar)
		})
	})

	Describe("creating deployment", func() {
		When("creating the deployment is successful", func() {
			BeforeEach(func() {
				fakeV7Actor.CreateDeploymentReturns(
					"some-deployment-guid",
					v7action.Warnings{"some-deployment-warning"},
					nil,
				)
			})

			It("waits for the app to start", func() {
				Expect(fakeV7Actor.PollStartForRollingCallCount()).To(Equal(1))
				givenAppGUID, givenDeploymentGUID, noWait := fakeV7Actor.PollStartForRollingArgsForCall(0)
				Expect(givenAppGUID).To(Equal("some-app-guid"))
				Expect(givenDeploymentGUID).To(Equal("some-deployment-guid"))
				Expect(noWait).To(Equal(false))
			})

			It("returns errors and warnings", func() {
				Expect(returnedPushPlan).To(Equal(paramPlan))
				Expect(executeErr).NotTo(HaveOccurred())
				Expect(warnings).To(ConsistOf("some-deployment-warning"))
			})

			It("records deployment events", func() {
				Expect(events).To(ConsistOf(StartingDeployment, WaitingForDeployment))
			})
		})

		When("creating the package errors", func() {
			var someErr error

			BeforeEach(func() {
				someErr = errors.New("failed to create deployment")

				fakeV7Actor.CreateDeploymentReturns(
					"",
					v7action.Warnings{"some-deployment-warning"},
					someErr,
				)
			})

			It("does not wait for the app to start", func() {
				Expect(fakeV7Actor.PollStartForRollingCallCount()).To(Equal(0))
			})

			It("returns errors and warnings", func() {
				Expect(returnedPushPlan).To(Equal(paramPlan))
				Expect(executeErr).To(MatchError(someErr))
				Expect(warnings).To(ConsistOf("some-deployment-warning"))
			})

			It("records deployment events", func() {
				Expect(events).To(ConsistOf(StartingDeployment))
			})
		})
	})

	Describe("waiting for app to start", func() {
		When("the the polling is successful", func() {
			BeforeEach(func() {
				fakeV7Actor.PollStartForRollingReturns(v7action.Warnings{"some-poll-start-warning"}, nil)
			})

			It("returns warnings and unchanged push plan", func() {
				Expect(returnedPushPlan).To(Equal(paramPlan))
				Expect(warnings).To(ConsistOf("some-poll-start-warning"))
			})

			It("records deployment events", func() {
				Expect(events).To(ConsistOf(StartingDeployment, WaitingForDeployment))
			})
		})

		When("the the polling returns an error", func() {
			var someErr error

			BeforeEach(func() {
				someErr = errors.New("app failed to start")
				fakeV7Actor.PollStartForRollingReturns(v7action.Warnings{"some-poll-start-warning"}, someErr)
			})

			It("returns errors and warnings", func() {
				Expect(warnings).To(ConsistOf("some-poll-start-warning"))
				Expect(executeErr).To(MatchError(someErr))
			})

			It("records deployment events", func() {
				Expect(events).To(ConsistOf(StartingDeployment, WaitingForDeployment))
			})
		})

		When("the noWait flag is set", func() {
			BeforeEach(func() {
				paramPlan.NoWait = true
			})

			It("passes in the noWait flag", func() {
				_, _, noWait := fakeV7Actor.PollStartForRollingArgsForCall(0)
				Expect(noWait).To(Equal(true))
			})
		})
	})
})
