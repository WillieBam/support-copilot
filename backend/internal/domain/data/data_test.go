package data_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/google/uuid"

	"github.com/WillieBam/support_copilot/backend/internal/domain/data"
	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/WillieBam/support_copilot/backend/types/responses"
)

var _ = Describe("Domain Data Serialisation", func() {
	Context("Alert Data Operations", func() {
		It("should parse alert args successfully", func() {
			raw := `{"alert_id": "12345"}`
			alertID, err := data.ParseAlertArgs(raw)
			Expect(err).To(BeNil())
			Expect(alertID).To(Equal("12345"))
		})

		It("should parse alert args with alert info id successfully", func() {
			raw := `{"alert": {"id": "12345"}}`
			alertID, err := data.ParseAlertArgs(raw)
			Expect(err).To(BeNil())
			Expect(alertID).To(Equal("12345"))
		})

		It("should return error on invalid alert args", func() {
			raw := `invalid json`
			_, err := data.ParseAlertArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should parse alert metrics successfully", func() {
			raw := `{"cpu_usage": 85.5}`
			metrics, err := data.ParseAlertMetrics(raw)
			Expect(err).To(BeNil())
			Expect(metrics.CpuUsage).To(Equal(85.5))
		})

		It("should return error on invalid alert metrics json", func() {
			raw := `invalid json`
			_, err := data.ParseAlertMetrics(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should marshal alerts to json", func() {
			alertID := uuid.New()
			alerts := []*models.Alert{
				{
					ID:           alertID,
					ResourceInfo: `{"service": "auth-service"}`,
					AlertInfo:    `{"severity": "critical"}`,
					ReceivedAt:   time.Now(),
				},
			}
			jsonStr, err := data.MarshalAlerts(alerts)
			Expect(err).To(BeNil())
			Expect(jsonStr).To(ContainSubstring("auth-service"))
		})

		It("should unmarshal alert section json", func() {
			sec, err := data.UnmarshalAlertSection(`{"id":"ALT-1","severity":"CRITICAL"}`)
			Expect(err).To(BeNil())
			Expect(sec.ID).To(Equal("ALT-1"))
			Expect(sec.Severity).To(Equal("CRITICAL"))
			Expect(sec.Message).To(BeEmpty())
		})

		It("should unmarshal resource section json", func() {
			sec, err := data.UnmarshalResourceSection(`{"service":"report-svc","environment":"prod"}`)
			Expect(err).To(BeNil())
			Expect(sec.Service).To(Equal("report-svc"))
			Expect(sec.Environment).To(Equal("prod"))
			Expect(sec.Cluster).To(BeEmpty())
		})

		It("should unmarshal full alert record with 5 sections", func() {
			alertID := uuid.New()
			model := &models.Alert{
				ID:              alertID,
				ReceivedAt:      time.Now(),
				AlertInfo:       `{"id":"ALT-100","severity":"CRITICAL"}`,
				ResourceInfo:    `{"service":"payment-svc"}`,
				Metrics:         `{"cpu_usage":91.4}`,
				BusinessContext: `{"business_service":"IBG"}`,
				Metadata:        `{"version":"1.0"}`,
			}
			rec, err := data.UnmarshalAlertRecord(model)
			Expect(err).To(BeNil())
			Expect(rec.ID).To(Equal(alertID.String()))
			Expect(rec.Resource.Service).To(Equal("payment-svc"))
			Expect(rec.Alert.Severity).To(Equal("CRITICAL"))
			Expect(*rec.Metrics.CPUUsage).To(Equal(91.4))
			Expect(rec.BusinessContext.BusinessService).To(Equal("IBG"))
			Expect(rec.Metadata.Version).To(Equal("1.0"))
		})


		It("should marshal validation result to json", func() {
			res := &responses.CombinedValidationResult{
				AlertID:     "alert-123",
				ServiceName: "payment",
			}
			jsonStr, err := data.MarshalValidationResult(res)
			Expect(err).To(BeNil())
			Expect(jsonStr).To(ContainSubstring("payment"))
		})

		It("should unmarshal alerts json string", func() {
			jsonStr := `[{"id":"a1","service_name":"auth","severity":"high","received_at":"2026-08-07T10:00:00Z"}]`
			records, err := data.UnmarshalAlerts(jsonStr)
			Expect(err).To(BeNil())
			Expect(records).To(HaveLen(1))
			Expect(records[0].ServiceName).To(Equal("auth"))
		})

		It("should return error on invalid unmarshal alerts json", func() {
			jsonStr := `invalid json`
			_, err := data.UnmarshalAlerts(jsonStr)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("Incident Data Operations", func() {
		It("should parse get incident args", func() {
			raw := `{"incident_id": "inc-1"}`
			args, err := data.ParseGetIncidentArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.IncidentID).To(Equal("inc-1"))
		})

		It("should return error on invalid get incident args", func() {
			raw := `invalid json`
			_, err := data.ParseGetIncidentArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should parse list incidents args", func() {
			raw := `{"team_id": "team-1"}`
			args, err := data.ParseListIncidentsArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.TeamID).To(Equal("team-1"))
		})

		It("should return error on invalid list incidents args", func() {
			raw := `invalid json`
			_, err := data.ParseListIncidentsArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should unmarshal incidents json", func() {
			jsonStr := `[{"id":"inc-1","title":"Outage"}]`
			records, err := data.UnmarshalIncidents(jsonStr)
			Expect(err).To(BeNil())
			Expect(records).To(HaveLen(1))
			Expect(records[0].Title).To(Equal("Outage"))
		})

		It("should return error on invalid unmarshal incidents json", func() {
			jsonStr := `invalid json`
			_, err := data.UnmarshalIncidents(jsonStr)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("Runbook Data Operations", func() {
		It("should parse create runbook args", func() {
			raw := `{"title": "Restart Pod", "content": "steps"}`
			args, err := data.ParseCreateRunbookArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.Title).To(Equal("Restart Pod"))
		})

		It("should return error on invalid create runbook args", func() {
			raw := `invalid json`
			_, err := data.ParseCreateRunbookArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should parse update runbook args", func() {
			raw := `{"runbook_id": "rb-1", "title": "New Title"}`
			args, err := data.ParseUpdateRunbookArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.RunbookID).To(Equal("rb-1"))
		})

		It("should return error on invalid update runbook args", func() {
			raw := `invalid json`
			_, err := data.ParseUpdateRunbookArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should parse deprecate runbook args", func() {
			raw := `{"runbook_id": "rb-1"}`
			args, err := data.ParseDeprecateRunbookArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.RunbookID).To(Equal("rb-1"))
		})

		It("should return error on invalid deprecate runbook args", func() {
			raw := `invalid json`
			_, err := data.ParseDeprecateRunbookArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should parse get runbook args", func() {
			raw := `{"runbook_id": "rb-1"}`
			args, err := data.ParseGetRunbookArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.RunbookID).To(Equal("rb-1"))
		})

		It("should return error on invalid get runbook args", func() {
			raw := `invalid json`
			_, err := data.ParseGetRunbookArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should parse list runbooks args", func() {
			raw := `{"team_id": "t-1", "status": "active"}`
			args, err := data.ParseListRunbooksArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.TeamID).To(Equal("t-1"))
		})

		It("should return error on invalid list runbooks args", func() {
			raw := `invalid json`
			_, err := data.ParseListRunbooksArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should parse link alert args", func() {
			raw := `{"alert_id": "a-1", "incident_id": "i-1"}`
			args, err := data.ParseLinkAlertArgs(raw)
			Expect(err).To(BeNil())
			Expect(args.AlertID).To(Equal("a-1"))
		})

		It("should return error on invalid link alert args", func() {
			raw := `invalid json`
			_, err := data.ParseLinkAlertArgs(raw)
			Expect(err).NotTo(BeNil())
		})

		It("should marshal link result", func() {
			str, err := data.MarshalLinkResult("a-1", "i-1")
			Expect(err).To(BeNil())
			Expect(str).To(ContainSubstring(`"status":"success"`))
		})

		It("should unmarshal runbooks json", func() {
			jsonStr := `[{"id":"rb-1","title":"Memory Leak Fix"}]`
			records, err := data.UnmarshalRunbooks(jsonStr)
			Expect(err).To(BeNil())
			Expect(records).To(HaveLen(1))
			Expect(records[0].Title).To(Equal("Memory Leak Fix"))
		})

		It("should return error on invalid unmarshal runbooks json", func() {
			jsonStr := `invalid json`
			_, err := data.UnmarshalRunbooks(jsonStr)
			Expect(err).NotTo(BeNil())
		})
	})
})
