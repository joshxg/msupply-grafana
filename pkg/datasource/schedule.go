package datasource

import (
	"database/sql"

	"excel-report-email-scheduler/pkg/ereserror"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/pkg/errors"
)

func NewSchedule(ID string, interval int, nextReportTime int, name string, description string, lookback string, reportGroupID string, time string, day int, dateFormat string, datePosition string, status string) Schedule {
	schedule := Schedule{
		ID:             ID,
		Interval:       interval,
		NextReportTime: nextReportTime,
		Name:           name,
		Description:    description,
		Lookback:       lookback,
		ReportGroupID:  reportGroupID,
		Time:           time,
		Day:            day,
		PanelDetails:   []ReportContent{},
		DateFormat:     dateFormat,
		DatePosition:   datePosition,
		Status:         status,
	}
	return schedule
}

func (datasource *MsupplyEresDatasource) GetSchedules() ([]Schedule, error) {
	frame := trace()
	sqlClient, err := datasource.NewSqlClient()
	if err != nil {
		err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not open database")
		return nil, err
	}
	defer sqlClient.Db.Close()

	var schedules []Schedule

	rows, err := sqlClient.Db.Query("SELECT * FROM Schedule")
	if err != nil {
		err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not get schedule list")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ID, Name, Description, ReportGroupID, Time, Lookback, DateFormat, DatePosition, Status string
		var Day, Interval, NextReportTime int

		err = rows.Scan(&ID, &Interval, &NextReportTime, &Name, &Description, &Lookback, &ReportGroupID, &Time, &Day, &DateFormat, &DatePosition, &Status)
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not scan schedule rows")
			return nil, err
		}

		reportContent, err := datasource.GetReportContent(ID)
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not get panel details")
			return nil, err
		}

		schedule := Schedule{
			ID:             ID,
			Interval:       Interval,
			NextReportTime: NextReportTime,
			Name:           Name,
			Description:    Description,
			Lookback:       Lookback,
			ReportGroupID:  ReportGroupID,
			Time:           Time,
			Day:            Day,
			PanelDetails:   reportContent,
			DateFormat:     DateFormat,
			DatePosition:   DatePosition,
			Status:         Status,
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

func (datasource *MsupplyEresDatasource) GetSchedule(id string) (*Schedule, error) {
	frame := trace()
	db, err := sql.Open("sqlite", datasource.DataPath)
	defer db.Close()
	if err != nil {
		log.DefaultLogger.Error("GetSchedule: sql.Open(): ", err.Error())
		return nil, err
	}

	var schedules []Schedule

	rows, err := db.Query("SELECT * FROM Schedule where id=?", id)
	defer rows.Close()
	if err != nil {
		log.DefaultLogger.Error("GetSchedules: db.Query(): ", err.Error())
		return nil, err
	}

	for rows.Next() {
		var ID, Name, Description, ReportGroupID, Time, Lookback, DateFormat, DatePosition, Status string
		var Day, Interval, NextReportTime int

		err = rows.Scan(&ID, &Interval, &NextReportTime, &Name, &Description, &Lookback, &ReportGroupID, &Time, &Day, &DateFormat, &DatePosition, &Status)
		if err != nil {
			log.DefaultLogger.Error("GetSchedules: rows.Scan(): ", err.Error())
			return nil, err
		}

		reportContent, err := datasource.GetReportContent(ID)
		if err != nil {
			err = ereserror.New(500, errors.Wrap(err, frame.Function), "Could not get panel details")
			return nil, err
		}

		schedule := Schedule{
			ID:             id,
			Interval:       Interval,
			NextReportTime: NextReportTime,
			Name:           Name,
			Description:    Description,
			Lookback:       Lookback,
			ReportGroupID:  ReportGroupID,
			Time:           Time,
			Day:            Day,
			PanelDetails:   reportContent,
			DateFormat:     DateFormat,
			DatePosition:   DatePosition,
			Status:         Status,
		}
		schedules = append(schedules, schedule)
	}

	if len(schedules) > 0 {
		return &schedules[0], nil
	} else {
		return nil, errors.New("no schedule found")
	}

}

func (datasource *MsupplyEresDatasource) UpdateSchedule(id string, status string, schedule Schedule) (*Schedule, error) {
	db, err := sql.Open("sqlite", datasource.DataPath)
	defer db.Close()
	if err != nil {
		log.DefaultLogger.Error(": sql.Open()", err.Error())
		return nil, err
	}

	stmt, err := db.Prepare("UPDATE Schedule SET nextReportTime = ?, interval = ?, name = ?, description = ?, lookback = ?, reportGroupID = ?, time = ?, day = ?, dateFormat = ?, datePosition = ?, status = ? where id = ?")
	if err != nil {
		log.DefaultLogger.Error("UpdateSchedule: db.Prepare()", err.Error())
		return nil, err
	}

	schedule.UpdateNextReportTime()

	_, err = stmt.Exec(schedule.NextReportTime, schedule.Interval, schedule.Name, schedule.Description, schedule.Lookback, schedule.ReportGroupID, schedule.Time, schedule.Day, schedule.DateFormat, schedule.DatePosition, status, id)
	defer stmt.Close()
	if err != nil {
		log.DefaultLogger.Error("UpdateSchedule: stmt.Exec()", err.Error())
		return nil, err
	}

	return &schedule, nil
}

func (datasource *MsupplyEresDatasource) UpdateScheduleProgess(id string, status string, schedule Schedule) (*Schedule, error) {
	db, err := sql.Open("sqlite", datasource.DataPath)
	defer db.Close()
	if err != nil {
		log.DefaultLogger.Error("UpdateSchedule: sql.Open()", err.Error())
		return nil, err
	}

	stmt, err := db.Prepare("UPDATE Schedule SET nextReportTime = ?, interval = ?, name = ?, description = ?, lookback = ?, reportGroupID = ?, time = ?, day = ?, dateFormat = ?, datePosition = ?, status = ? where id = ?")
	log.DefaultLogger.Info("anil test executing update code for status",status)
	log.DefaultLogger.Info("anil test executing update code for schedule",schedule)
	
	if err != nil {
		log.DefaultLogger.Error("UpdateSchedule: db.Prepare()", err.Error())
		return nil, err
	}

	_, err = stmt.Exec(schedule.NextReportTime, schedule.Interval, schedule.Name, schedule.Description, schedule.Lookback, schedule.ReportGroupID, schedule.Time, schedule.Day, schedule.DateFormat, schedule.DatePosition, status, id)
	defer stmt.Close()
	if err != nil {
		log.DefaultLogger.Error("UpdateSchedule: stmt.Exec()", err.Error())
		return nil, err
	}

	return &schedule, nil
}

func (datasource *MsupplyEresDatasource) DeleteSchedule(id string) error {
	db, err := sql.Open("sqlite", datasource.DataPath)
	defer db.Close()
	if err != nil {
		log.DefaultLogger.Error("DeleteSchedule: sql.Open()", err.Error())
		return err
	}

	stmt, err := db.Prepare("DELETE FROM Schedule WHERE id = ?")
	defer stmt.Close()
	if err != nil {
		log.DefaultLogger.Error("DeleteSchedule: db.Prepare()1", err.Error())
		return err
	}

	_, err = stmt.Exec(id)
	if err != nil {
		log.DefaultLogger.Error("DeleteSchedule: stmt.Exec()1", err.Error())
		return err
	}

	stmt, err = db.Prepare("DELETE FROM ReportContent WHERE scheduleID = ?")
	if err != nil {
		log.DefaultLogger.Error("DeleteSchedule: db.Prepare()2", err.Error())
		return err
	}

	stmt.Exec(id)
	if err != nil {
		log.DefaultLogger.Error("DeleteSchedule: stmt.Exec()2", err.Error())
		return err
	}

	return nil
}

func (datasource *MsupplyEresDatasource) OverdueSchedules() ([]Schedule, error) {
	db, err := sql.Open("sqlite", datasource.DataPath)
	defer db.Close()
	if err != nil {
		log.DefaultLogger.Error("OverdueSchedules: sql.Open", err.Error())
		return nil, err
	}

	rows, err := db.Query("SELECT * FROM Schedule WHERE strftime(\"%s\", \"now\") > nextReportTime and status = \"success\"")
	log.DefaultLogger.Info("executing SELECT * FROM Schedule WHERE strftime(\"%s\", \"now\") > nextReportTime and status = success")

	if err != nil {
		log.DefaultLogger.Error("OverdueSchedules: db.Query", err.Error())
		return nil, err
	}

	var schedules []Schedule
	for rows.Next() {
		var ID, Name, Description, ReportGroupID, Time, Lookback, DateFormat, DatePosition, Status string
		var Day, Interval, NextReportTime int
		err = rows.Scan(&ID, &Interval, &NextReportTime, &Name, &Description, &Lookback, &ReportGroupID, &Time, &Day, &DateFormat, &DatePosition, &Status)
		if err != nil {
			log.DefaultLogger.Error("OverdueSchedules: sql.Open", err.Error())
			return nil, err
		}
		schedules = append(schedules, NewSchedule(ID, Interval, NextReportTime, Name, Description, Lookback, ReportGroupID, Time, Day, DateFormat, DatePosition, Status))
	}

	return schedules, nil
}
