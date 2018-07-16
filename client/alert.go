package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/suzuki-shunsuke/go-graylog"
)

// GetAlerts returns all alerts.
func (client *Client) GetAlerts(
	skip, limit int,
) ([]graylog.Alert, int, *ErrorInfo, error) {
	return client.GetAlertsContext(context.Background(), skip, limit)
}

// GetAlertsContext returns all alerts with a context.
func (client *Client) GetAlertsContext(ctx context.Context, skip, limit int) (
	[]graylog.Alert, int, *ErrorInfo, error,
) {
	body := &graylog.AlertsBody{}
	v := url.Values{
		"skip":  []string{strconv.Itoa(skip)},
		"limit": []string{strconv.Itoa(limit)},
	}
	u := fmt.Sprintf("%s?%s", client.Endpoints().Alerts(), v.Encode())
	ei, err := client.callGet(ctx, u, nil, body)
	return body.Alerts, body.Total, ei, err
}
