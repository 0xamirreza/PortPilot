// Package events names audit actions.
package events

const (
	ActionAppStart    = "APPLICATION_START"
	ActionSettings    = "SETTINGS_UPDATE"
	ActionFWRead      = "FIREWALL_READ"
	ActionFWApply     = "FIREWALL_APPLY"
	ActionFWDelete    = "FIREWALL_DELETE"
	ActionFWReconcile = "FIREWALL_RECONCILE"
	ActionService     = "SERVICE_ACTION"
	ActionSignal      = "PROCESS_SIGNAL"
	ResultSuccess     = "SUCCESS"
	ResultFailed      = "FAILED"
	ResultDenied      = "DENIED"
)
