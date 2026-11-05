package model

import "time"

type Course struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	CourseCode  string    `json:"course_code" gorm:"uniqueIndex;size:20;not null"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Credits     float64   `json:"credits"`
	Department  string    `json:"department" gorm:"size:100"`
	Teacher     string    `json:"teacher" gorm:"size:50"`
	Semester    string    `json:"semester" gorm:"size:20"`
	Description string    `json:"description" gorm:"size:500"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
