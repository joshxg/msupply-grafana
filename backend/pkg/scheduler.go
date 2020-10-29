package main

import (
	"fmt"
	"net/smtp"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type EmailConfig struct {
	email    string
	password string
}

type SmtpServer struct {
	host string
	port string
}

func (smtpServer *SmtpServer) Address() string {
	return smtpServer.host + ":" + smtpServer.port
}

// Possibly take the email config, an attachment and some array of recipients?
func sendEmail() {

	// TODO: Password and email from params. Leave hard coded for now
	// for dev'in
	from := "testemailsussol@gmail.com"

	// This password is an app-specific password. The real password
	// to the account is kathmandu312. Seems to require me to generate
	// and use an app-specific password. :shrug:
	password := "ybtkmpesjptowmru"
	to := []string{
		"griffinjoshua5@gmail.com",
	}

	// TODO: Host and port need to be added to datasource config?
	smtpServer := SmtpServer{host: "smtp.gmail.com", port: "587"}

	// TODO: Subject, message to be added to datasource config? Or schedule config?
	message := []byte("Reports from mSupply")

	// I don't really know what I'm doing with this auth.
	// PlainAuth works and reading the docs it seems to fail
	// if not using TLS. So I guess it's probably OK.
	auth := smtp.PlainAuth("", from, password, smtpServer.host)

	err := smtp.SendMail(smtpServer.Address(), auth, from, to, message)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Email Sent!")
}

// TODO: Handle cases where an email config doesn't exist or invalid etc.
// Could send an email to somewhere else, sussol support or something
// if this is the case?
func getEmailConfig(sqlite *SQLiteDatasource) EmailConfig {
	return sqlite.getEmailConfig()
}

func getSchedulesToReportOn(sqlite *SQLiteDatasource) {
	// const schedules = sqlite.getSchedules()
	// const timeNow = Date.now()
	// const schedulesToReportOn = schedules.filter(schedule => schedule.nextReportTime < timeNow)

	// return schedulesToReportOn
}

func getUsersToReportTo(sqlite *SQLiteDatasource) {

	// const schedules = getSchedulesToReportOn()

	// Get the report groups assigned to the schedules that are being reported on.
	// const reportGroups = sqlite.getReportGroups(schedules)

	// Find the report group members (Join table between report group and user)
	// const reportGroupMembers = sqlite.getReportGroupMembers(reportGroups)

	// const users = need to do an http call here to the postgres db with the userIDs from the report group member records.

	// const lookup = create some sort of lookup table with a shape:
	// { {scheduleID}: [user emails] }

	// return users
}

func getPanelsForSchedules(sqlite *SQLiteDatasource) {
	// Get the schedules
	// const schedules = getSchedulesToReportOn()

	// Create a lookup in the shape:
	// { {scheduleID}: [panelIDs] }
	// const lookup = schedules.reduce((acc, value) => ({...acc, [value.id]: sqlite.getPanels(scheduleID)}) , {})
}

func createReports(sqlite *SQLiteDatasource) {

	// Get the panels for each schedule
	// const panelLookup = getPanelsForSchedules()

	// for each schedule, for each panel, query for the panel data
	// and create an excel file where each tab is a panel table.
	// save in a new lookup the shape:
	// { {scheduleID}: excelFilePath }
}

func sendEmails(sqlite *SQLiteDatasource) {
	// const attachments = createReports()
	// const recipients = getUsersToReportTo()

	// We could do some intersection and send a single
	// email with multiple attachments to some users
	// or combine the attachments or something like
	// that.. but seems too hard basket

	// for each key, value in attachments
	// user the key to lookup in the recipients
	// lookup to get the recipient emails.
	// and call sendEmail(config, recipients, attachment)
}

func cleanup() {
	// Not sure exactly but need to:
	// 1. Delete all temp attachment files?
	// 2. Set the new times on all of the schedules
}

func getScheduler(sqlite *SQLiteDatasource) func() {
	return func() {
		sendEmail()
		log.DefaultLogger.Info("Scheduler!")
	}
}