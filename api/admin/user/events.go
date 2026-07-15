package user

// EventUserCreated is the event type published when a user is created. Peer
// services subscribe to "<prefix>.user.created" (see lib/events) to react —
// e.g. sending a welcome email or provisioning downstream resources.
const EventUserCreated = "user.created"

// UserCreatedEvent is the payload carried by EventUserCreated. Keep it to the
// stable, non-sensitive fields other services legitimately need; never emit
// secrets such as password hashes.
type UserCreatedEvent struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}
