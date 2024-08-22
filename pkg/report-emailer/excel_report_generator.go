package reportEmailer

import (
	"errors"
	"excel-report-email-scheduler/pkg/api"
	"excel-report-email-scheduler/pkg/auth"
	"fmt"
	_ "image/png"
	"math"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

var CELL_REF_REG = regexp.MustCompile("([A-Za-z]+)|([0-9]+)")

func intToCol(i int) string {
	i += 1
	result := ""
	for i > 0 {
		rem := i % 26
		if rem == 0 {
			result = "Z" + result
			i = i/26 - 1
		} else {
			result = fmt.Sprintf("%c", 65+rem-1) + result
			i = i / 26
		}
	}
	return result
}

func NewReport(id string, name string, templatePath string) *Report {
	return &Report{id: id, name: name, templatePath: templatePath}
}

func (r *Report) openTemplate() error {
	f, err := excelize.OpenFile(r.templatePath)
	if err != nil {
		log.DefaultLogger.Error("Could not open template: ", err.Error())
		return err
	}

	r.file = f

	return nil
}

func (r *Report) placeholderCellRef(sheetName string, placeholder string) ([]string, error) {
	refs := r.file.SearchSheet(sheetName, placeholder)

	if len(refs) == 0 {
		err := errors.New("Could not find cell reference for: " + sheetName + " - " + placeholder)
		log.DefaultLogger.Error("placeholderCellRef", err.Error())
		return nil, err
	}

	return refs, nil
}

func (r *Report) placeholderRowRef(sheetName string, placeholder string) (int, error) {
	refs := r.file.SearchSheet(sheetName, placeholder)

	if len(refs) == 0 {
		err := errors.New("Could not find cell reference for: " + sheetName + " - " + placeholder)
		log.DefaultLogger.Error("placeholderRowRef", err.Error())
		return 0, err
	}

	split := CELL_REF_REG.FindAllString(refs[0], 2)
	idx, err := strconv.Atoi(split[1])
	if err != nil {
		log.DefaultLogger.Error("placeholderRowRef: Atoi: " + err.Error())
		return 0, err
	}

	return idx, nil
}

func (r *Report) writeTitle(sheetName string) error {
	refs, err := r.placeholderCellRef(sheetName, "{{title}}")
	if err != nil {
		log.DefaultLogger.Error("writeTitle: placeholderCellRef: " + err.Error())
		return err
	}

	r.file.SetCellValue(sheetName, refs[0], sheetName)

	return nil
}

func (r *Report) writeDate(sheetName string) error {
	refs, err := r.placeholderCellRef(sheetName, "{{date}}")
	if err != nil {
		log.DefaultLogger.Error("writeDate: ", err.Error())
		return err
	}

	r.file.SetCellValue(sheetName, refs[0], time.Now().Format("Mon Jan 2 15:04:05"))

	return nil
}

func toDateString(value interface{}) (string, bool) {
	var date string

	// for some queries dates are returned as float64 values
	// this is a quick check to see if they might be dates
	if timestamp, ok := value.(float64); ok {
		// 2000/01/01 < timestamp < 2500/01/01
		if timestamp > 946684800000 && timestamp < 16725225600000 {
			unix := time.Unix(int64(timestamp/1000), 0)
			date = unix.Format("2006/01/02")

			return date, true
		}
	}

	// for others they are showing as strings
	// even though both are date type in postgres
	if datestring, ok := value.(string); ok {
		matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, datestring)
		if matched {
			if parsed, err := time.Parse(time.RFC3339, datestring); err == nil {
				date = parsed.Format("2006/01/02")
				return date, true
			}
		}

	}

	return date, false
}

func (r *Report) writeCell(sheetName string, cellRef string, value interface{}) {
	if date, ok := toDateString(value); ok {
		value = date
		if style, err := r.file.NewStyle(`{"number_format": 14,  "alignment": { "horizontal": "right", "vertical": "center" }}`); err == nil {
			r.file.SetCellStyle(sheetName, cellRef, cellRef, style)
		}
	}
	if _, ok := value.(string); ok {
		r.file.SetCellValue(sheetName, cellRef, value)
	} else if boolean, ok := value.(bool); ok {
		r.file.SetCellBool(sheetName, cellRef, boolean)
	} else {
		if style, err := r.file.NewStyle(`{"number_format": 0, "alignment": { "vertical": "center" }}`); err == nil {
			r.file.SetCellStyle(sheetName, cellRef, cellRef, style)
		}
		r.file.SetCellValue(sheetName, cellRef, value)
	}
}

func (r *Report) SetSheets(panels []api.TablePanel) {
	r.sheets = panels
}

func (r *Report) writeHeaders(sheetName string, columns []api.Column) error {
	idx, err := r.placeholderRowRef(sheetName, "{{headers}}")

	style := r.file.GetCellStyle(sheetName, r.createCellRef(0, idx))

	if err != nil {
		log.DefaultLogger.Error("writeHeaders: ", err.Error())
		return err
	}

	if len(columns) > 0 {
		for i, column := range columns {
			cellRef := r.createCellRef(i, idx)
			r.file.SetCellStyle(sheetName, cellRef, cellRef, style)
			r.writeCell(sheetName, cellRef, column.Text)
		}
	} else {
		cellRef := r.createCellRef(0, idx)
		r.writeCell(sheetName, cellRef, "")
	}

	return nil
}

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func (r *Report) createDuplicateTableRows(sheetName string, numberToDuplicate int) (int, error) {
	idx, err := r.placeholderRowRef(sheetName, "{{rows}}")
	if err != nil {
		log.DefaultLogger.Error("createDuplicateTableRows: " + err.Error())
		return 0, err
	}

	duplicateNumber := Min(numberToDuplicate, 500)
	i := 0
	for i < duplicateNumber {
		r.file.DuplicateRow(sheetName, idx)
		i += 1
	}

	return idx, nil
}

func (r *Report) createCellRef(columnNumber int, rowNumber int) string {
	return intToCol(columnNumber) + strconv.Itoa(rowNumber)
}

func (r *Report) writeRows(sheetName string, rows [][]interface{}) error {
	idx, err := r.placeholderRowRef(sheetName, "{{rows}}")
	if err != nil {
		log.DefaultLogger.Error("writeRows: placeholderRowRef: " + err.Error())
		return err
	}

	style := r.file.GetCellStyle(sheetName, r.createCellRef(0, idx))
	if len(rows) > 0 {
		for _, row := range rows {
			for j, value := range row {
				cellRef := r.createCellRef(j, idx)
				r.file.SetCellStyle(sheetName, cellRef, cellRef, style)
				if value == nil {
					r.writeCell(sheetName, cellRef, "\n")
				} else {
					r.writeCell(sheetName, cellRef, value)
				}

			}
			idx += 1
		}
	} else {
		cellRef := r.createCellRef(0, idx)
		r.writeCell(sheetName, cellRef, "No data")
	}

	return nil
}

func (r *Report) setColumnWidths(sheetName string, columns []api.Column, rows [][]interface{}) error {

	headerFontCorrectionFactor := 1.4
	maximumContentLengths := make(map[int]float64)

	for columnNumber, column := range columns {
		maximumContentLengths[columnNumber] = headerFontCorrectionFactor * (2 + float64(len(column.Text)))
	}

	if len(rows) == 0 {
		return nil
	}

	for _, row := range rows {
		for i, value := range row {
			if value != nil {
				if timestamp, ok := value.(float64); ok {
					// adding 3 to the length, as we are formatting with 2 decimal places
					maximumContentLengths[i] = math.Max(maximumContentLengths[i], float64(3+len(strconv.FormatFloat(timestamp, 'f', -1, 64))))

					// if a date, then reformat as the appropriate date string
					if date, ok := toDateString(value); ok {
						value = date
					}
				}

				if _, ok := value.(string); ok {
					maximumContentLengths[i] = math.Max(maximumContentLengths[i], float64(len(value.(string))))
				}
			}
		}
	}

	for columnNumber, _ := range rows {
		r.file.SetColWidth(sheetName, intToCol(columnNumber), intToCol(columnNumber), 1.5+maximumContentLengths[columnNumber])
	}

	return nil
}

func (r *Report) Write(auth auth.AuthConfig) error {
	log.DefaultLogger.Info(fmt.Sprintf("Starting to create report %s...", r.id))
	if r.file == nil {
		if err := r.openTemplate(); err != nil {
			return err
		}
	}

	for _, s := range r.sheets {
		log.DefaultLogger.Info(fmt.Sprintf("Creating new sheet %s", s.Title))
		sIdx := r.file.NewSheet(s.Title)

		if err := r.file.CopySheet(1, sIdx); err != nil {
			log.DefaultLogger.Error("Write: copySheet: " + err.Error())
			return err
		}

		if err := s.GetData(auth); err != nil {
			log.DefaultLogger.Error("Write: GetData: " + err.Error())
			return err
		}

		if err := r.writeTitle(s.Title); err != nil {
			log.DefaultLogger.Error("Write: writeTitle: " + err.Error())
			return err
		}

		if err := r.writeDate(s.Title); err != nil {
			log.DefaultLogger.Error("Write: writeDate: " + err.Error())
			return err
		}

		if err := r.writeHeaders(s.Title, s.Columns); err != nil {
			log.DefaultLogger.Error("Write: writeHeaders: " + err.Error())
			return err
		}

		if err := r.writeRows(s.Title, s.Rows); err != nil {
			log.DefaultLogger.Error("Write: writeRows: " + err.Error())
			return err
		}

		if err := r.setColumnWidths(s.Title, s.Columns, s.Rows); err != nil {
			log.DefaultLogger.Error("Write: setColumnWidths: " + err.Error())
			return err
		}
	}

	r.file.DeleteSheet("templateSheet")

	log.DefaultLogger.Info("Saving report...")

	savePath := GetFilePath(r.name)
	if err := r.file.SaveAs(savePath); err != nil {
		log.DefaultLogger.Error("Write: ", err.Error())
	}

	log.DefaultLogger.Info(fmt.Sprintf("Report finished! %s (%s) :tada", r.id, savePath))

	return nil
}

func (r *Reporter) ExportPanel(authConfig *auth.AuthConfig, datasourceID int, dashboardID string, panelID int, query string, title string) (string, error) {

	dashboard, err := api.NewDashboard(authConfig, dashboardID, "", "", datasourceID)
	if err != nil {
		log.DefaultLogger.Error("Reporter.ExportPanel: NewDashboard: " + err.Error())
		return "", err
	}

	panel := dashboard.Panel(panelID)
	if panel == nil {
		log.DefaultLogger.Error("Reporter.ExportPanel: panel is nil")
		return "", errors.New(fmt.Sprintf("panel with ID %d cannot be found. DashboardID: %s, datasourceID: %d", panelID, dashboardID, datasourceID))
	}

	panel.SetSql(query)
	log.DefaultLogger.Debug("Reporter.ExportPanel: Query=" + query)
	panel.SetTitle(title)

	reportSheetPanels := []api.TablePanel{*panel}
	report := r.CreateNewReport(strconv.Itoa(panelID), panel.Title)
	report.SetSheets(reportSheetPanels)

	err = report.Write(*authConfig)
	if err != nil {
		log.DefaultLogger.Error("ReportEmailer.createReports: report.Write: " + err.Error())
		return "", err
	}

	return panel.Title + ".xlsx", nil
}

func GetFilePath(fileName string) string {

	var runningOS string
	var filePath string

	runningOS = runtime.GOOS

	if runningOS == "windows" {
		filePath = filepath.Join("..", "data", fileName+".xlsx")
	} else {
		filePath = filepath.Join("/var/lib/grafana/plugins", "data", fileName+".xlsx")
	}

	log.DefaultLogger.Debug("mSupply App: ReportFilePath=" + filePath)
	return filePath
}

func NewReporter(templatePath string) *Reporter {
	return &Reporter{templatePath: templatePath}
}

func (r *Reporter) CreateNewReport(scheduleID string, scheduleName string) *Report {
	report := NewReport(scheduleID, scheduleName, r.templatePath)
	return report
}

func (re *Reporter) GetFilePath(fileName string) string {
	return GetFilePath(fileName)
}
