package reportEmailer

import (
	"excel-report-email-scheduler/pkg/api"
	"excel-report-email-scheduler/pkg/auth"
	"excel-report-email-scheduler/pkg/datasource"
	"fmt"
	"os"

	"github.com/bugsnag/bugsnag-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/robfig/cron"
)

func NewReportEmailer(datasource *datasource.MsupplyEresDatasource) *ReportEmailer {
	re := ReportEmailer{datasource: datasource, inProgress: false}
	re.Init()
	return &re
}

func (re *ReportEmailer) Init() {
	// Try to send reports on loading
	re.CreateReports()

	// Set up scheduler which will try to send reports every 10 minutes
	c := cron.New()
	c.AddFunc("@every 5m", func() {
		re.CreateReports()
	})

	c.Start()
}

func (re *ReportEmailer) configs() (*auth.AuthConfig, *auth.EmailConfig, int, error) {
	settings, err := re.datasource.NewSettings()
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.configs: NewSettings: " + err.Error())
		return nil, nil, 0, err
	}

	authConfig, err := auth.NewAuthConfig(settings)
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.configs: NewSettings: " + err.Error())
		return nil, nil, 0, err
	}

	emailConfig, err := auth.NewEmailConfig(settings)
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.configs: NewEmailConfig: " + err.Error())
		return nil, nil, 0, err
	}

	return authConfig, emailConfig, settings.DatasourceID, nil
}

func (re *ReportEmailer) cleanup(schedules []datasource.Schedule) {
	log.DefaultLogger.Info("Starting Clean up...")
	for _, schedule := range schedules {

		log.DefaultLogger.Info(fmt.Sprintf("Deleting %s.xlsx...", schedule.Name))
		schedularFileName := GetFormattedFileName(schedule.Name, schedule.DateFormat, schedule.DatePosition)
		path := GetFilePath(schedularFileName)
		err := os.Remove(path)
		if err != nil {
			// Failure case shouldn't be much of a problem since we're using the schedule ID for the report name, at the moment
			// as it will just write to the same file and not create infinitely many if deleting always fails.
			log.DefaultLogger.Error(fmt.Sprintf("Could not delete %s... : %s.xlsx", schedule.Name, err.Error()))
		}

		schedule.UpdateNextReportTime()
	    log.DefaultLogger.Info("anil test cleanup schedule id "+schedule.ID + "to status success")
		re.datasource.UpdateSchedule(schedule.ID, "success", schedule)
	}

	re.inProgress = false
}

func (re *ReportEmailer) CreateReport(schedule datasource.Schedule, authConfig *auth.AuthConfig, datasourceID int, em Emailer) error {

	log.DefaultLogger.Debug("ReportEmailer.createReport: start")
	emails := make(map[string][]string)
	panels := make(map[string][]api.TablePanel)

	reportGroup, err := re.datasource.ReportGroupFromSchedule(schedule)
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.createReport: ReportGroupFromSchedule: " + err.Error())
		return err
	} else {
		log.DefaultLogger.Debug("ReportEmailer.createReport: ReportGroupFromSchedule:", reportGroup)
	}

	userIDs, err := re.datasource.GroupMemberUserIDs(reportGroup)
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.createReport: GroupMemberUserIDs: " + err.Error())
		return err
	} else {
		log.DefaultLogger.Debug("ReportEmailer.createReport: GroupMemberUserIDs:", userIDs)
	}

	emailsFromUsers, err := api.GetEmails(*authConfig, userIDs, datasourceID)
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.createReport: emailsFromUsers: " + err.Error())
		return err
	} else {
		log.DefaultLogger.Debug("ReportEmailer.createReport: GetEmails:", emailsFromUsers)
	}

	emails[schedule.ID] = emailsFromUsers

	reportContent, err := re.datasource.GetReportContent(schedule.ID)
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.createReport: GetReportContent: " + err.Error())
		return err
	} else {
		log.DefaultLogger.Debug("ReportEmailer.createReport: GetReportContent:", reportContent)
	}

	panels[schedule.ID] = []api.TablePanel{}

	for _, content := range reportContent {
		lookback := content.Lookback
		to := "now"
		from := lookback

		dashboard, err := api.NewDashboard(authConfig, content.DashboardID, from, to, datasourceID)

		if err != nil {
			log.DefaultLogger.Error("ReportEmailer.createReport: NewDashboard: " + err.Error())
			return err
		} else {
			log.DefaultLogger.Debug("ReportEmailer.createReport: NewDashboard:", dashboard)
		}

		panel := dashboard.Panel(content.PanelID)
		if panel != nil {
			panel.PrepSql(dashboard.Variables, content.Variables)
			panels[schedule.ID] = append(panels[schedule.ID], *panel)
		}
	}

	if len(panels) == 0 {
		log.DefaultLogger.Info("ReportEmailer.createReports - no panels! quitting report as nothing to do")
		return nil
	}

	templatePath := GetFilePath("template")
	log.DefaultLogger.Debug("ReportEmailer.createReport: templatePath:", templatePath)
	reporter := NewReporter(templatePath)

	log.DefaultLogger.Debug("ReportEmailer.createReport: panels being used: ", panels)
	for scheduleID, reportSheetPanels := range panels {
		schedule, err := re.datasource.GetSchedule(scheduleID)

		if err != nil {
			log.DefaultLogger.Error("ReportEmailer: GetSchedule: Could not create report to send.", err.Error())
			bugsnag.Notify(err)
		} else {
			log.DefaultLogger.Debug(fmt.Sprintf("ReportEmailer.createReport: schedule: %s", schedule.Name))
			schedularFileName := GetFormattedFileName(schedule.Name, schedule.DateFormat, schedule.DatePosition)
			log.DefaultLogger.Info("schedularFileName", schedularFileName)
			report := reporter.CreateNewReport(scheduleID, schedularFileName)
			report.SetSheets(reportSheetPanels)
			err := report.Write(*authConfig)
			if err != nil {
				log.DefaultLogger.Error("ReportEmailer.createReports: report.Write: " + err.Error())
				return err
			}
		}
	}

	for scheduleID, recipientEmails := range emails {
		schedule, err := re.datasource.GetSchedule(scheduleID)
		if err != nil {
			log.DefaultLogger.Error("ReportEmailer: GetSchedule: Could not create report to send.", err.Error())
			bugsnag.Notify(err)
		} else {
			schedularFileName := GetFormattedFileName(schedule.Name, schedule.DateFormat, schedule.DatePosition)
			attachmentPath := GetFilePath(schedularFileName)
			em.BulkCreateAndSend(attachmentPath, recipientEmails, schedularFileName, schedule.Description)
		}
	}

	return nil
	// re.cleanup(schedules)
}

func (re *ReportEmailer) CreateReports() {
	log.DefaultLogger.Info("Creating Reports...")
	re.inProgress = true

	authConfig, emailConfig, datasourceID, err := re.configs()
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.createReports: re.configs: " + err.Error())
		return
	}

	em := NewEmailSender(emailConfig)

	schedules, err := re.datasource.OverdueSchedules()
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.createReports: OverdueSchedules: " + err.Error())
		return
	}

	if len(schedules) > 0 {
		log.DefaultLogger.Info("Found schedules which are overdue...")
		for _, schedule := range schedules {
			log.DefaultLogger.Info(fmt.Sprintf("- %s : %s", schedule.Name, schedule.Description))
		}
	} else {
		log.DefaultLogger.Info("No schedules are overdue...")
		return
	}

	emails := make(map[string][]string)
	panels := make(map[string][]api.TablePanel)
	for _, schedule := range schedules {
		reportGroup, err := re.datasource.ReportGroupFromSchedule(schedule)
		log.DefaultLogger.Info("anil test updating schedule id "+schedule.ID + "to status progress")
		re.datasource.UpdateScheduleProgess(schedule.ID, "progress", schedule)

		if err != nil {
			log.DefaultLogger.Error("ReportEmailer.createReports: ReportGroupFromSchedule: " + err.Error())
			return
		}

		userIDs, err := re.datasource.GroupMemberUserIDs(reportGroup)
		if err != nil {
			log.DefaultLogger.Error("ReportEmailer.createReports: GroupMemberUserIDs: " + err.Error())
			return
		}

		emailsFromUsers, err := api.GetEmails(*authConfig, userIDs, datasourceID)
		if err != nil {
			log.DefaultLogger.Error("ReportEmailer.createReports: emailsFromUsers: " + err.Error())
			return
		}

		emails[schedule.ID] = emailsFromUsers

		reportContent, err := re.datasource.GetReportContent(schedule.ID)
		if err != nil {
			log.DefaultLogger.Error("ReportEmailer.createReports: GetReportContent: " + err.Error())
			return
		}

		panels[schedule.ID] = []api.TablePanel{}

		for _, content := range reportContent {
			lookback := content.Lookback
			to := "now"
			from := lookback

			dashboard, err := api.NewDashboard(authConfig, content.DashboardID, from, to, datasourceID)

			if err != nil {
				log.DefaultLogger.Error("ReportEmailer.createReports: NewDashboard: " + err.Error())
				return
			}

			panel := dashboard.Panel(content.PanelID)

			if panel != nil {
				panel.PrepSql(dashboard.Variables, content.Variables)
				panels[schedule.ID] = append(panels[schedule.ID], *panel)
			}

		}
	}

	templatePath := GetFilePath("template")
	reporter := NewReporter(templatePath)

	for scheduleID, reportSheetPanels := range panels {
		schedule, err := re.datasource.GetSchedule(scheduleID)
		if err != nil {
			log.DefaultLogger.Error("ReportEmailer: GetSchedule: Could not create report to send.", err.Error())
			bugsnag.Notify(err)
		} else {
			schedularFileName := GetFormattedFileName(schedule.Name, schedule.DateFormat, schedule.DatePosition)
			log.DefaultLogger.Info("schedularFileName", schedularFileName)
			report := reporter.CreateNewReport(scheduleID, schedularFileName)

			report.SetSheets(reportSheetPanels)
			err := report.Write(*authConfig)
			if err != nil {
				log.DefaultLogger.Error("ReportEmailer.createReports: report.Write: " + err.Error())
				return
			}
		}

	}

	for scheduleID, recipientEmails := range emails {
		schedule, err := re.datasource.GetSchedule(scheduleID)
		if err != nil {
			log.DefaultLogger.Error("ReportEmailer: GetSchedule: Could not create report to send.", err.Error())
			bugsnag.Notify(err)
		} else {
			schedularFileName := GetFormattedFileName(schedule.Name, schedule.DateFormat, schedule.DatePosition)
			attachmentPath := GetFilePath(schedularFileName)
			log.DefaultLogger.Info("scheduleDescription", schedule.Description)
			em.BulkCreateAndSend(attachmentPath, recipientEmails, schedularFileName, schedule.Description)
		}
	}

	re.cleanup(schedules)

}
