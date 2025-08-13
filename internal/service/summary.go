package service

import (
	"gorinha/internal/database"
	"gorinha/internal/models"
	"time"
)

func GetSummary(
	db *database.Store,
	fromStr,
	toStr string,
) (summary map[string]*models.Summary, err error) {
	summary = map[string]*models.Summary{
		"default":  {TotalRequests: 0, TotalAmount: 0},
		"fallback": {TotalRequests: 0, TotalAmount: 0},
	}

	from := int64(0)
	to := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()

	if fromStr != "" {
		t, err := time.Parse(time.RFC3339Nano, fromStr)
		if err == nil {
			from = t.UnixNano()
		}
	}

	if toStr != "" {
		t, err := time.Parse(time.RFC3339Nano, toStr)
		if err == nil {
			to = t.UnixNano()
		}
	}

	data, err := db.RangeQuery(0, from, to)

	if err != nil {
		return
	}

	summary["default"].TotalRequests = len(data)
	for _, amount := range data {
		summary["default"].TotalAmount += amount

	}

	data, err = db.RangeQuery(2, from, to)

	if err != nil {
		return
	}

	summary["fallback"].TotalRequests = len(data)
	for _, amount := range data {
		summary["fallback"].TotalAmount += amount

	}
	return

}
