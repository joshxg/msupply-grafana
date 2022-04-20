package datasource

import (
	"database/sql"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type ReportGroupMembership struct {
	ID            string `json:"id"`
	UserID        string `json:"userID"`
	ReportGroupID string `json:"reportGroupID"`
}

func ReportGroupMembershipFields() string {
	return "\n{\n\tID string\n\tUserID string\nReportGroupID string\n}"
}

func NewReportGroupMembership(ID string, userID string, reportGroupID string) *ReportGroupMembership {
	return &ReportGroupMembership{ID: ID, UserID: userID, ReportGroupID: reportGroupID}
}

func (datasource *MsupplyEresDatasource) GroupMemberUserIDs(reportGroup ReportGroup) ([]string, error) {
	db, err := sql.Open("sqlite", datasource.DataPath)
	if err != nil {
		log.DefaultLogger.Error("GroupMemberUserIDs: sql.Open", err.Error())
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM ReportGroupMembership WHERE reportGroupID = ?", reportGroup.ID)
	if err != nil {
		log.DefaultLogger.Error("GroupMemberUserIDs: db.Query()", err.Error())
		return nil, err
	}

	var memberships []ReportGroupMembership
	for rows.Next() {
		var ID, UserID, ReportGroupID string
		err = rows.Scan(&ID, &UserID, &ReportGroupID)
		if err != nil {
			log.DefaultLogger.Error("GroupMemberUserIDs: rows.Scan(): ", err.Error())
			return nil, err
		}
		membership := ReportGroupMembership{ID, UserID, ReportGroupID}
		memberships = append(memberships, membership)
	}

	var userIDs []string
	for _, member := range memberships {
		userIDs = append(userIDs, member.UserID)
	}

	return userIDs, nil
}