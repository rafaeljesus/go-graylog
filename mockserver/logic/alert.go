package logic

import (
	"github.com/suzuki-shunsuke/go-graylog"
)

// GetAlerts returns a list of alerts.
func (lgc *Logic) GetAlerts(since, limit int) ([]graylog.Alert, int, int, error) {
	arr, total, err := lgc.store.GetAlerts(since, limit)
	if err != nil {
		return nil, 0, 500, err
	}
	return arr, total, 200, nil
}
