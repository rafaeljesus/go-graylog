package endpoint

// Alerts returns an Alert API's endpoint url.
func (ep *Endpoints) Alerts() string {
	return ep.alerts.String()
}
