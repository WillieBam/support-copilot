package errors_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	customErrors "github.com/WillieBam/support_copilot/backend/utils/errors"
)

var _ = Describe("Custom Domain Errors", func() {
	Context("Auth Errors", func() {
		It("should define ErrInvalidPasswordComplexity with non empty message", func() {
			Expect(customErrors.ErrInvalidPasswordComplexity).NotTo(BeNil())
			Expect(customErrors.ErrInvalidPasswordComplexity.Error()).To(ContainSubstring("password must be between 6 and 8 characters"))
		})
	})

	Context("Team Errors", func() {
		It("should define team sentinel errors with valid messages", func() {
			Expect(customErrors.ErrTeamNameRequired.Error()).To(Equal("team name is required"))
			Expect(customErrors.ErrTeamNameTooLong.Error()).To(Equal("team name must be 20 characters or less"))
			Expect(customErrors.ErrUnauthorizedTeamOp.Error()).To(Equal("unauthorized team operation: owner permission required"))
			Expect(customErrors.ErrSuperAdminRequired.Error()).To(Equal("unauthorized operation: super_admin scope required to delete a team"))
			Expect(customErrors.ErrUserNotInTeam.Error()).To(Equal("user is not a member of this team"))
			Expect(customErrors.ErrInvalidIncidentStatus.Error()).To(Equal("invalid incident status: must be OPEN, IN_PROGRESS, RESOLVED, or CLOSED"))
			Expect(customErrors.ErrIncidentNotFound.Error()).To(Equal("incident not found"))
			Expect(customErrors.ErrInstructionTooShort.Error()).To(Equal("instruction details must be at least 30 characters long"))
		})
	})

	Context("Dashboard Errors", func() {
		It("should define dashboard sentinel errors with valid messages", func() {
			Expect(customErrors.ErrInvalidTimeframe.Error()).To(Equal("invalid timeframe: must be day, month, or year"))
			Expect(customErrors.ErrInvalidSLATarget.Error()).To(Equal("sla_target_minutes must be a positive integer"))
			Expect(customErrors.ErrDashboardUnauthorized.Error()).To(Equal("unauthorized: must be a team member to access dashboard analytics"))
		})
	})
})
