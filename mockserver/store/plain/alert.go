package plain

import (
	"github.com/suzuki-shunsuke/go-graylog"
)

// GetAlerts returns Alerts.
func (store *Store) GetAlerts(since, limit int) ([]graylog.Alert, int, error) {
	// TODO treat since parameter
	store.imutex.RLock()
	defer store.imutex.RUnlock()
	size := len(store.alerts)
	if size == 0 {
		return nil, 0, nil
	}
	if limit > size {
		limit = size
	}

	arr := make([]graylog.Alert, limit)
	i := 0
	for _, a := range store.alerts {
		arr[i] = a
		i++
		if i == limit {
			break
		}
	}
	return arr, limit, nil
}
