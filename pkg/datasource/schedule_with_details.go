package datasource

import (
	"excel-report-email-scheduler/pkg/ereserror"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/pkg/errors"
)

type Schedule struct {
	ID             string          `json:"id"`
	Interval       int             `json:"interval"`
	NextReportTime int             `json:"nextReportTime"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Lookback       string          `json:"lookback,string"`
	ReportGroupID  string          `json:"reportGroupID"`
	Time           string          `json:"time"`
	Day            int             `json:"day"`
	PanelDetails   []ReportContent `json:"panelDetails"`
	DateFormat     string          `json:"dateFormat"`
	DatePosition   string          `json:"datePosition"`
	Status         string          `json:"status"`
}

type ReportContent struct {
	ID          string `json:"id"`
	ScheduleID  string `json:"scheduleID"`
	PanelID     int    `json:"panelID"`
	DashboardID string `json:"dashboardID"`
	Lookback    string `json:"lookback"`
	Variables   string `json:"variables"`
}

func (datasource *MsupplyEresDatasource) CreateScheduleWithDetails(scheduleWithDetails Schedule) (*Schedule, error) {
	frame := trace()
	sqlClient, err := datasource.NewSqlClient()
	if err != nil {
		err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not open database")
		return nil, err
	}
	defer sqlClient.Db.Close()

	if scheduleWithDetails.ID == "" {
		stmt, err := sqlClient.Db.Prepare("INSERT INTO Schedule (id, nextReportTime, interval, name, description, lookback,reportGroupID,time,day,dateFormat, datePosition, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?) RETURNING *")
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not create schedule record")
			return nil, err
		}
		defer stmt.Close()

		scheduleWithDetails.ID = uuid.New().String()

		scheduleWithDetails.UpdateNextReportTime()
		scheduleWithDetails.Status = "success"
		
		_, err = stmt.Exec(scheduleWithDetails.ID, scheduleWithDetails.NextReportTime, scheduleWithDetails.Interval, scheduleWithDetails.Name, scheduleWithDetails.Description, scheduleWithDetails.Lookback, scheduleWithDetails.ReportGroupID, scheduleWithDetails.Time, scheduleWithDetails.Day, scheduleWithDetails.DateFormat, scheduleWithDetails.DatePosition, scheduleWithDetails.Status)
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not create schedule record")
			return nil, err
		}
	} else {
		stmt, err := sqlClient.Db.Prepare("UPDATE Schedule SET nextReportTime = ?, interval = ?, name = ?, description = ?, lookback = ?, reportGroupID = ?, time = ?, day = ?, dateFormat = ?, datePosition = ?, status = ? where id = ? RETURNING *")
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not update schedule record")
			return nil, err
		}
		defer stmt.Close()

		scheduleWithDetails.UpdateNextReportTime()

		_, err = stmt.Exec(scheduleWithDetails.NextReportTime, scheduleWithDetails.Interval, scheduleWithDetails.Name, scheduleWithDetails.Description, scheduleWithDetails.Lookback, scheduleWithDetails.ReportGroupID, scheduleWithDetails.Time, scheduleWithDetails.Day, scheduleWithDetails.DateFormat, scheduleWithDetails.DatePosition, scheduleWithDetails.Status, scheduleWithDetails.ID)
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not update schedule record")
			return nil, err
		}

		err = datasource.DeleteReportContentByScheduleID(scheduleWithDetails.ID)
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not update schedule record")
			return nil, err
		}
	}

	var reportContents []ReportContent
	for _, paneDetail := range scheduleWithDetails.PanelDetails {
		newUuid := uuid.New().String()
		reportContent := ReportContent{ID: newUuid, ScheduleID: scheduleWithDetails.ID, PanelID: paneDetail.PanelID, DashboardID: paneDetail.DashboardID, Lookback: paneDetail.Lookback, Variables: paneDetail.Variables}
		reportContents = append(reportContents, reportContent)
	}

	_, err = datasource.CreateReportContents(reportContents)
	if err != nil {
		err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not update schedule report content records")
		return nil, err
	}

	return &scheduleWithDetails, nil
}

func (schedule *Schedule) UpdateNextReportTime() {
	now := time.Now()
	daysOffset := 1
	scheduleDays := 1
	if schedule.Day > 0 {
		scheduleDays = schedule.Day
	}

	reportTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	// there is probably a better way to parse the time string, it didn't work for me though
	timeOfDay, err := time.Parse(time.RFC3339, "1970-01-01T"+schedule.Time+":00+00:00")
	if err == nil {
		log.DefaultLogger.Info(fmt.Sprintf("Adding time of '%s' to '%s'", schedule.Time, reportTime))
		reportTime = time.Date(reportTime.Year(), reportTime.Month(), reportTime.Day(), timeOfDay.Hour(), timeOfDay.Minute(), 0, 0, now.Location())
	}

	// for the intervals using x Day of y, remove the current Day value
	daysOffset = (-1 * int(reportTime.Day())) + scheduleDays

	switch schedule.Interval {
	case 5: //yearly
		if daysOffset > 365 {
			reportTime = reportTime.AddDate(2, 0, -reportTime.Day())
		} else {
			reportTime = reportTime.AddDate(1, 0, daysOffset)
		}
	case 4: // quarterly
		if daysOffset > 93 {
			reportTime = reportTime.AddDate(0, 6, -reportTime.Day())
		} else {
			reportTime = reportTime.AddDate(0, 3, daysOffset)
		}
	case 3: // monthly
		if daysOffset > 31 {
			reportTime = reportTime.AddDate(0, 2, -reportTime.Day())
		} else {
			reportTime = reportTime.AddDate(0, 1, daysOffset)
		}
	case 2: // fortnightly
		if scheduleDays == int(reportTime.Day()) {
			reportTime = reportTime.AddDate(0, 0, 14)
		} else {
			daysToAdd := (scheduleDays - int(reportTime.Day()) + 14) % 14
			reportTime = reportTime.AddDate(0, 0, daysToAdd)
		}

	case 1: // weekly
		if scheduleDays == int(reportTime.Weekday()) {
			reportTime = reportTime.AddDate(0, 0, 7)
		} else {
			daysToAdd := (scheduleDays - int(reportTime.Weekday()) + 7) % 7
			reportTime = reportTime.AddDate(0, 0, daysToAdd)
		}

	default: // 0 == daily
		if reportTime.Unix() < now.Unix() {
			// run tomorrow
			reportTime = reportTime.AddDate(0, 0, 1)
		}
	}
	schedule.NextReportTime = int(reportTime.Unix())
	log.DefaultLogger.Info(fmt.Sprintf("Setting time of schedule '%s' to '%s'", schedule.Name, reportTime))
}
