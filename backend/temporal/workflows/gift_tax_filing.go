package workflows

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/wealth"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// GiftTaxFilingWorkflow orchestrates Form 709 preparation and filing
func GiftTaxFilingWorkflow(ctx workflow.Context, input GiftTaxFilingInput) (*GiftTaxFilingResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting gift tax filing workflow", "familyID", input.FamilyID, "taxYear", input.TaxYear)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &GiftTaxFilingResult{
		FamilyID: input.FamilyID,
		TaxYear:  input.TaxYear,
		Forms:    []wealth.Form709{},
	}

	// Step 1: Get all gifts for the tax year requiring Form 709
	var giftsRequiringFiling []wealth.GiftForFiling
	err := workflow.ExecuteActivity(ctx, "GetGiftsRequiringForm709Activity", wealth.GetGiftsRequiringForm709Input{
		FamilyID: input.FamilyID,
		TaxYear:  input.TaxYear,
	}).Get(ctx, &giftsRequiringFiling)
	if err != nil {
		return nil, err
	}

	if len(giftsRequiringFiling) == 0 {
		logger.Info("No gifts requiring Form 709 filing")
		result.Message = "No Form 709 filing required for this tax year"
		return result, nil
	}

	logger.Info("Found gifts requiring filing", "count", len(giftsRequiringFiling))

	// Step 2: Group gifts by donor (one Form 709 per donor)
	giftsByDonor := make(map[string][]wealth.GiftForFiling)
	for _, gift := range giftsRequiringFiling {
		giftsByDonor[gift.DonorMemberID] = append(giftsByDonor[gift.DonorMemberID], gift)
	}

	// Step 3: Prepare Form 709 for each donor
	for donorID, donorGifts := range giftsByDonor {
		var form709 wealth.Form709
		err := workflow.ExecuteActivity(ctx, "PrepareForm709Activity", wealth.PrepareForm709Input{
			FamilyID:      input.FamilyID,
			DonorMemberID: donorID,
			TaxYear:       input.TaxYear,
			Gifts:         donorGifts,
		}).Get(ctx, &form709)
		if err != nil {
			logger.Error("Failed to prepare Form 709", "donorID", donorID, "error", err)
			continue
		}

		result.Forms = append(result.Forms, form709)
	}

	logger.Info("Prepared Forms 709", "count", len(result.Forms))

	// Step 4: Calculate total gift and GST taxes
	for i := range result.Forms {
		var taxAmounts wealth.TaxAmounts
		err := workflow.ExecuteActivity(ctx, "CalculateGiftAndGSTTaxActivity", wealth.CalculateGiftAndGSTTaxInput{
			Form709: result.Forms[i],
		}).Get(ctx, &taxAmounts)
		if err != nil {
			logger.Warn("Failed to calculate taxes", "formID", result.Forms[i].FormID, "error", err)
			continue
		}

		result.Forms[i].TotalGiftTax = taxAmounts.GiftTax
		result.Forms[i].TotalGSTTax = taxAmounts.GSTTax
		result.Forms[i].TotalTaxDue = taxAmounts.TotalTaxDue
	}

	// Step 5: Generate PDF for each form (if requested)
	if input.GeneratePDF {
		for i := range result.Forms {
			var pdfPath string
			err := workflow.ExecuteActivity(ctx, "GenerateForm709PDFActivity", wealth.GenerateForm709PDFInput{
				Form709: result.Forms[i],
			}).Get(ctx, &pdfPath)
			if err != nil {
				logger.Warn("Failed to generate PDF", "formID", result.Forms[i].FormID, "error", err)
				continue
			}

			result.Forms[i].PDFPath = pdfPath
		}
	}

	// Step 6: If auto-file enabled, submit to IRS
	if input.AutoFile {
		for i := range result.Forms {
			var confirmationNumber string
			err := workflow.ExecuteActivity(ctx, "FileForm709ElectronicallyActivity", wealth.FileForm709Input{
				Form709: result.Forms[i],
			}).Get(ctx, &confirmationNumber)
			if err != nil {
				logger.Error("Failed to e-file Form 709", "formID", result.Forms[i].FormID, "error", err)
				result.Forms[i].FilingStatus = "FAILED"
				result.Forms[i].FilingError = err.Error()
				continue
			}

			result.Forms[i].FilingStatus = "FILED"
			result.Forms[i].FilingConfirmation = confirmationNumber
			result.Forms[i].FilingDate = time.Now().Format("2006-01-02")

			// Update gift records to mark as filed
			err = workflow.ExecuteActivity(ctx, "MarkGiftsAsFiledActivity", wealth.MarkGiftsAsFiledInput{
				FormID:     result.Forms[i].FormID,
				FilingDate: time.Now(),
			}).Get(ctx, nil)
			if err != nil {
				logger.Warn("Failed to update gift records", "error", err)
			}
		}
	} else {
		// Mark as prepared but not filed
		for i := range result.Forms {
			result.Forms[i].FilingStatus = "PREPARED"
		}
	}

	// Step 7: Send notification to advisor
	err = workflow.ExecuteActivity(ctx, "SendFilingNotificationActivity", wealth.SendFilingNotificationInput{
		FamilyID:  input.FamilyID,
		TaxYear:   input.TaxYear,
		Forms:     result.Forms,
		AutoFiled: input.AutoFile,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to send notification", "error", err)
	}

	// Step 8: Calculate due dates and set reminders
	dueDate := time.Date(input.TaxYear+1, 4, 15, 0, 0, 0, 0, time.UTC)
	if time.Now().After(dueDate) {
		result.IsOverdue = true
		result.DaysOverdue = int(time.Since(dueDate).Hours() / 24)
	} else {
		result.DueDate = dueDate.Format("2006-01-02")
		result.DaysUntilDue = int(time.Until(dueDate).Hours() / 24)
	}

	result.FormsGenerated = len(result.Forms)
	result.Message = "Form 709 filing workflow complete"

	logger.Info("Gift tax filing workflow complete",
		"forms", result.FormsGenerated,
		"autoFiled", input.AutoFile,
		"dueDate", result.DueDate)

	return result, nil
}

// GiftTaxFilingInput is the workflow input
type GiftTaxFilingInput struct {
	FamilyID    string `json:"family_id"`
	TaxYear     int    `json:"tax_year"`
	GeneratePDF bool   `json:"generate_pdf"`
	AutoFile    bool   `json:"auto_file"` // If true, electronically file with IRS
}

// GiftTaxFilingResult is the workflow result
type GiftTaxFilingResult struct {
	FamilyID       string           `json:"family_id"`
	TaxYear        int              `json:"tax_year"`
	Forms          []wealth.Form709 `json:"forms"`
	FormsGenerated int              `json:"forms_generated"`
	DueDate        string           `json:"due_date,omitempty"`
	DaysUntilDue   int              `json:"days_until_due,omitempty"`
	IsOverdue      bool             `json:"is_overdue"`
	DaysOverdue    int              `json:"days_overdue,omitempty"`
	Message        string           `json:"message"`
}
