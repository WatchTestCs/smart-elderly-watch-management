package socialrepo

import (
	"context"

	"suining-care/internal/db"
)

// Comment is the public notice-comment read model.
type Comment struct {
	SaidID     *int64  `json:"saidId"`
	Said       *string `json:"said"`
	TextID     *int64  `json:"textId"`
	Back       *string `json:"back"`
	SaidTime   *string `json:"saidTime"`
	BackTime   *string `json:"backTime"`
	PersonName *string `json:"personName"`
	NiceCount  *int64  `json:"niceCount"`
	SaidPhone  *string `json:"saidPhone"`
	TouPic     *string `json:"touPic"`
	Good       *string `json:"good"`
}

func maskPhone(phone *string) *string {
	if phone == nil || len(*phone) != 11 {
		return nil
	}
	value := *phone
	masked := value[:3] + "****" + value[7:]
	return &masked
}

// ListComments returns a notice's comments and the viewer's own vote status.
// A zero noticeID selects all comments using the existing endpoint semantics.
func ListComments(ctx context.Context, store *db.Database, viewerPhone string, noticeID int64) ([]Comment, error) {
	query := `SELECT s.said_id, s.said, s.text_id, s.back,` +
		` TO_CHAR(s.said_time,'YYYY-MM-DD HH24:MI:SS.MS') AS saidTime,` +
		` TO_CHAR(s.back_time,'YYYY-MM-DD HH24:MI:SS.MS') AS backTime,` +
		` s.person_name, s.nice_count, s.said_phone, s.tou_pic,` +
		` CASE WHEN EXISTS (SELECT 1 FROM nice_count n WHERE n.said_id=s.said_id AND n.phone=$1) THEN '1' ELSE '0' END AS good` +
		` FROM said s`
	args := []any{viewerPhone}
	if noticeID != 0 {
		query += " WHERE s.text_id=$2"
		args = append(args, noticeID)
	}
	return db.QueryList(ctx, store, func(rows db.Rows) (Comment, error) {
		var item Comment
		err := rows.Scan(&item.SaidID, &item.Said, &item.TextID, &item.Back, &item.SaidTime, &item.BackTime,
			&item.PersonName, &item.NiceCount, &item.SaidPhone, &item.TouPic, &item.Good)
		if err == nil {
			item.SaidPhone = maskPhone(item.SaidPhone)
		}
		return item, err
	}, query, args...)
}
