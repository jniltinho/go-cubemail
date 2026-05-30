package model

// EventAttendee represents a participant invited to a calendar event.
// PartStat follows iCalendar values: NEEDS-ACTION, ACCEPTED, DECLINED, TENTATIVE.
type EventAttendee struct {
	ID       uint   `gorm:"primaryKey"`
	EventID  uint   `gorm:"index;not null"`
	Name     string `gorm:"size:255"`
	Email    string `gorm:"size:255;not null"`
	PartStat string `gorm:"size:20;default:'NEEDS-ACTION'"`
	Role     string `gorm:"size:30;default:'REQ-PARTICIPANT'"`
	RSVP     bool   `gorm:"default:true"`
}
