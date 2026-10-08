package calendar

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/hondyman/uisce/backend/internal/compliance/engine"
)

// StatutoryScheduleRecord matches compliance.statutory_calendar_schedule schema.
type StatutoryScheduleRecord struct {
	ScheduleCode      string
	Title             string
	Description       string
	Jurisdiction      string
	Regulation        string
	DeadlineType      string
	RecurrencePattern string
	MonthOffsets      pq.Int64Array
	DayOfMonth        int
	CutoffTime        string
	LegalCitation     string
	RuleIDs           pq.StringArray
	Severity          string
}

// Service provides compliance calendar calculation, statutory deadline schedules, and dynamic breach filings.
type Service struct {
	db              *sql.DB
	tradingCalendar *engine.TradingCalendar
}

// NewService creates a new compliance calendar Service.
func NewService(db *sql.DB) *Service {
	return &Service{
		db:              db,
		tradingCalendar: engine.NewTradingCalendar(),
	}
}

// GetCalendarEvents returns all compliance deadlines within [from, to] for the tenant.
func (s *Service) GetCalendarEvents(ctx context.Context, filter CalendarFilter) ([]ComplianceCalendarEvent, error) {
	refDay := filter.ReferenceDay
	if refDay.IsZero() {
		refDay = time.Now().UTC()
	}

	fromDate := filter.FromDate
	if fromDate.IsZero() {
		fromDate = refDay.AddDate(0, -1, 0)
	}
	toDate := filter.ToDate
	if toDate.IsZero() {
		toDate = refDay.AddDate(0, 6, 0)
	}

	var events []ComplianceCalendarEvent

	// 1. Statutory Standing Schedules from Database
	statutory, err := s.loadStatutorySchedules(ctx, filter.TenantID, fromDate, toDate, refDay)
	if err == nil {
		events = append(events, statutory...)
	}

	// 2. Dynamic Regulatory Change Cases from Database
	if s.db != nil {
		regCases, err := s.queryRegulatoryCaseEvents(ctx, filter.TenantID, fromDate, toDate, refDay)
		if err == nil {
			events = append(events, regCases...)
		}

		// 3. Dynamic Surveillance Exception Statutory Deadlines from Database
		survEvents, err := s.querySurveillanceExceptionEvents(ctx, filter.TenantID, fromDate, toDate, refDay)
		if err == nil {
			events = append(events, survEvents...)
		}
	}

	// 4. Apply in-memory filtering
	var filtered []ComplianceCalendarEvent
	for _, e := range events {
		if filter.Jurisdiction != "" && filter.Jurisdiction != "ALL" && e.Jurisdiction != filter.Jurisdiction {
			continue
		}
		if filter.Regulation != "" && e.Regulation != filter.Regulation {
			continue
		}
		if filter.DeadlineType != "" && e.DeadlineType != filter.DeadlineType {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		filtered = append(filtered, e)
	}

	// Sort chronologically by DueDate and CutoffTime
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].DueDate == filtered[j].DueDate {
			return filtered[i].CutoffTime < filtered[j].CutoffTime
		}
		return filtered[i].DueDate < filtered[j].DueDate
	})

	return filtered, nil
}

// loadStatutorySchedules reads schedule definitions from compliance.statutory_calendar_schedule table.
func (s *Service) loadStatutorySchedules(ctx context.Context, tenantID uuid.UUID, from, to, refDay time.Time) ([]ComplianceCalendarEvent, error) {
	var records []StatutoryScheduleRecord

	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `
			SELECT schedule_code, title, description, jurisdiction, regulation, deadline_type,
			       recurrence_pattern, month_offsets, day_of_month, cutoff_time, legal_citation, rule_ids, severity
			FROM compliance.statutory_calendar_schedule
			WHERE is_active = true
			ORDER BY schedule_code
		`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var rec StatutoryScheduleRecord
				if err := rows.Scan(
					&rec.ScheduleCode, &rec.Title, &rec.Description, &rec.Jurisdiction, &rec.Regulation,
					&rec.DeadlineType, &rec.RecurrencePattern, &rec.MonthOffsets, &rec.DayOfMonth,
					&rec.CutoffTime, &rec.LegalCitation, &rec.RuleIDs, &rec.Severity,
				); err == nil {
					records = append(records, rec)
				}
			}
		}
	}

	// Fallback definitions if DB is unseeded/unit-test mode
	if len(records) == 0 {
		records = []StatutoryScheduleRecord{
			{
				ScheduleCode:      "SEC_13F_QUARTERLY",
				Title:             "SEC Form 13F Institutional Holdings Filing",
				Description:       "Mandatory quarterly filing for institutional investment managers with discretion over $100M+ in Section 13(f) securities.",
				Jurisdiction:      "US",
				Regulation:        "SEC Form 13F",
				DeadlineType:      "STATUTORY_FILING",
				RecurrencePattern: "QUARTERLY_45D",
				MonthOffsets:      pq.Int64Array{2, 5, 8, 11},
				DayOfMonth:        14,
				CutoffTime:        "17:30 EST",
				LegalCitation:     "Securities Exchange Act of 1934 Section 13(f)(1)",
				RuleIDs:           pq.StringArray{"SEC-13F-001"},
				Severity:          "HIGH",
			},
			{
				ScheduleCode:      "UK_TAKEOVER_RULE_8_3",
				Title:             "UK Takeover Panel Rule 8.3 Dealing Disclosure Review",
				Description:       "Public dealing disclosure deadline for interests in relevant securities of 1% or more during offer period.",
				Jurisdiction:      "UK",
				Regulation:        "Takeover Panel Rule 8.3",
				DeadlineType:      "DISCLOSURE_CUTOFF",
				RecurrencePattern: "MONTHLY_MID",
				MonthOffsets:      pq.Int64Array{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
				DayOfMonth:        15,
				CutoffTime:        "12:00 BST",
				LegalCitation:     "The City Code on Takeovers and Mergers, Rule 8.3",
				RuleIDs:           pq.StringArray{"UK-TAKEOVER-RULE83"},
				Severity:          "CRITICAL",
			},
		}
	}

	var events []ComplianceCalendarEvent
	startYear := from.Year() - 1
	endYear := to.Year() + 1

	for _, rec := range records {
		for year := startYear; year <= endYear; year++ {
			for _, m := range rec.MonthOffsets {
				due := time.Date(year, time.Month(m), rec.DayOfMonth, 17, 0, 0, 0, time.UTC)
				// Adjust for trading calendar if weekend/holiday
				for !s.tradingCalendar.IsTradingDay(due, rec.Jurisdiction) {
					due = due.AddDate(0, 0, -1)
				}

				if s.isBetween(due, from, to) {
					daysRem := s.calculateBusinessDaysRemaining(refDay, due, rec.Jurisdiction)
					events = append(events, ComplianceCalendarEvent{
						ID:              uuid.NewSHA1(uuid.NameSpaceDNS, []byte(fmt.Sprintf("%s-%d-%d-%s", rec.ScheduleCode, year, m, tenantID))),
						TenantID:        tenantID,
						EventCode:       fmt.Sprintf("%s_%d_%02d", rec.ScheduleCode, year, m),
						Title:           fmt.Sprintf("%s (%02d/%d)", rec.Title, m, year),
						Description:     rec.Description,
						Jurisdiction:    rec.Jurisdiction,
						Regulation:      rec.Regulation,
						DeadlineType:    rec.DeadlineType,
						DueDate:         due.Format("2006-01-02"),
						CutoffTime:      rec.CutoffTime,
						CutoffTimestamp: &due,
						DaysRemaining:   daysRem,
						Status:          s.resolveStatus(daysRem),
						Severity:        rec.Severity,
						LegalCitation:   rec.LegalCitation,
						RuleIDs:         rec.RuleIDs,
						Source:          "STATUTORY_SCHEDULE",
					})
				}
			}
		}
	}

	return events, nil
}

// queryRegulatoryCaseEvents fetches pending regulatory cases with DueAt dates.
func (s *Service) queryRegulatoryCaseEvents(ctx context.Context, tenantID uuid.UUID, from, to, refDay time.Time) ([]ComplianceCalendarEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, case_code, title, description, source, classification, due_at, status
		FROM compliance.regulatory_change_case
		WHERE due_at IS NOT NULL
		  AND due_at BETWEEN $1 AND $2
		  AND status != 'PUBLISHED'
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ComplianceCalendarEvent
	for rows.Next() {
		var id uuid.UUID
		var caseCode, title, desc, source, classification, status string
		var dueAt time.Time
		if err := rows.Scan(&id, &caseCode, &title, &desc, &source, &classification, &dueAt, &status); err != nil {
			continue
		}

		jurisdiction := "GLOBAL"
		switch source {
		case "SEC", "FINRA", "CFTC":
			jurisdiction = "US"
		case "FCA", "PRA", "TAKEOVER_PANEL":
			jurisdiction = "UK"
		case "ESMA", "EBA", "ECB":
			jurisdiction = "EU"
		}

		daysRem := s.calculateBusinessDaysRemaining(refDay, dueAt, jurisdiction)
		events = append(events, ComplianceCalendarEvent{
			ID:              id,
			TenantID:        tenantID,
			EventCode:       caseCode,
			Title:           fmt.Sprintf("Regulatory Change: %s (%s)", title, caseCode),
			Description:     desc,
			Jurisdiction:    jurisdiction,
			Regulation:      source,
			DeadlineType:    "REGULATORY_CHANGE",
			DueDate:         dueAt.Format("2006-01-02"),
			CutoffTime:      dueAt.Format("15:04 MST"),
			CutoffTimestamp: &dueAt,
			DaysRemaining:   daysRem,
			Status:          s.resolveStatus(daysRem),
			Severity:        "HIGH",
			Source:          "REGULATORY_CASE",
			SourceID:        id.String(),
			Metadata: map[string]any{
				"classification": classification,
				"case_status":    status,
			},
		})
	}

	return events, nil
}

// querySurveillanceExceptionEvents fetches active statutory disclosure breaches from post-trade surveillance.
func (s *Service) querySurveillanceExceptionEvents(ctx context.Context, tenantID uuid.UUID, from, to, refDay time.Time) ([]ComplianceCalendarEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, rule_id, rule_name, rule_pack, statutory_deadline_at, remediation_status, severity
		FROM compliance.surveillance_exception
		WHERE statutory_deadline_at IS NOT NULL
		  AND statutory_deadline_at BETWEEN $1 AND $2
		  AND remediation_status != 'RESOLVED'
		  AND tenant_id = $3
	`, from, to, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ComplianceCalendarEvent
	for rows.Next() {
		var id, accountID uuid.UUID
		var ruleID, ruleName, rulePack, remediationStatus, severity string
		var deadline time.Time
		if err := rows.Scan(&id, &accountID, &ruleID, &ruleName, &rulePack, &deadline, &remediationStatus, &severity); err != nil {
			continue
		}

		jurisdiction := "GLOBAL"
		if rulePack == "SEC" || rulePack == "1940_ACT" {
			jurisdiction = "US"
		} else if rulePack == "FCA" || rulePack == "UK_TAKEOVER" {
			jurisdiction = "UK"
		} else if rulePack == "UCITS" || rulePack == "ESMA" {
			jurisdiction = "EU"
		}

		daysRem := s.calculateBusinessDaysRemaining(refDay, deadline, jurisdiction)
		events = append(events, ComplianceCalendarEvent{
			ID:               id,
			TenantID:         tenantID,
			EventCode:        fmt.Sprintf("DISCLOSURE_%s", id.String()[:8]),
			Title:            fmt.Sprintf("Statutory Breach Disclosure: %s", ruleName),
			Description:      fmt.Sprintf("Mandatory regulatory filing triggered by threshold breach on rule %s for account %s.", ruleID, accountID),
			Jurisdiction:     jurisdiction,
			Regulation:       rulePack,
			DeadlineType:     "DISCLOSURE_CUTOFF",
			DueDate:          deadline.Format("2006-01-02"),
			CutoffTime:       deadline.Format("15:04 MST"),
			CutoffTimestamp:  &deadline,
			DaysRemaining:    daysRem,
			Status:           s.resolveStatus(daysRem),
			Severity:         severity,
			AffectedAccounts: []string{accountID.String()},
			RuleIDs:          []string{ruleID},
			Source:           "SURVEILLANCE_BREACH",
			SourceID:         id.String(),
			Metadata: map[string]any{
				"remediation_status": remediationStatus,
			},
		})
	}

	return events, nil
}

func (s *Service) isBetween(t, from, to time.Time) bool {
	tDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	fDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDate := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)
	return !tDate.Before(fDate) && !tDate.After(toDate)
}

func (s *Service) calculateBusinessDaysRemaining(refDay, target time.Time, jurisdiction string) int {
	refDate := time.Date(refDay.Year(), refDay.Month(), refDay.Day(), 0, 0, 0, 0, time.UTC)
	tgtDate := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, time.UTC)

	if refDate.Equal(tgtDate) {
		return 0
	}

	if refDate.Before(tgtDate) {
		days := 0
		curr := refDate
		for curr.Before(tgtDate) {
			curr = curr.AddDate(0, 0, 1)
			if s.tradingCalendar.IsTradingDay(curr, jurisdiction) {
				days++
			}
		}
		return days
	}

	// Overdue case: target is in the past
	days := 0
	curr := tgtDate
	for curr.Before(refDate) {
		curr = curr.AddDate(0, 0, 1)
		if s.tradingCalendar.IsTradingDay(curr, jurisdiction) {
			days--
		}
	}
	return days
}

func (s *Service) resolveStatus(daysRemaining int) string {
	if daysRemaining < 0 {
		return "OVERDUE"
	}
	if daysRemaining <= 5 {
		return "DUE_SOON"
	}
	return "UPCOMING"
}
