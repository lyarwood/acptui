package ambient_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/lyarwood/acptui/internal/ambient"
)

var _ = Describe("Session", func() {
	Describe("IsStartable", func() {
		DescribeTable("returns correct value for each phase",
			func(phase string, expected bool) {
				s := ambient.Session{Phase: phase}
				Expect(s.IsStartable()).To(Equal(expected))
			},
			Entry("Pending", "Pending", true),
			Entry("Stopped", "Stopped", true),
			Entry("Failed", "Failed", true),
			Entry("Completed", "Completed", true),
			Entry("empty", "", true),
			Entry("Running", "Running", false),
			Entry("Creating", "Creating", false),
			Entry("Stopping", "Stopping", false),
		)
	})

	Describe("Age", func() {
		It("returns UpdatedAt when available", func() {
			now := time.Now()
			created := now.Add(-1 * time.Hour)
			s := ambient.Session{
				CreatedAt: &created,
				UpdatedAt: &now,
			}
			Expect(s.Age()).To(Equal(now))
		})

		It("falls back to CreatedAt", func() {
			created := time.Now()
			s := ambient.Session{
				CreatedAt: &created,
			}
			Expect(s.Age()).To(Equal(created))
		})

		It("returns zero time when no timestamps", func() {
			s := ambient.Session{}
			Expect(s.Age().IsZero()).To(BeTrue())
		})
	})

	Describe("FlexMatch", func() {
		It("matches case-insensitively", func() {
			Expect(ambient.FlexMatch("Hello World", "hello")).To(BeTrue())
		})

		It("supports regex patterns", func() {
			Expect(ambient.FlexMatch("test-123", "test-\\d+")).To(BeTrue())
		})

		It("falls back to substring on invalid regex", func() {
			Expect(ambient.FlexMatch("hello[world", "[world")).To(BeTrue())
		})

		It("returns false when no match", func() {
			Expect(ambient.FlexMatch("hello", "goodbye")).To(BeFalse())
		})
	})
})
