package model

import "time"

type StudentStatus string

const (
	StudentStatusActive    StudentStatus = "在读"
	StudentStatusSuspended StudentStatus = "休学"
	StudentStatusGraduated StudentStatus = "毕业"
	StudentStatusDropped   StudentStatus = "退学"
)

type Student struct {
	ID             uint          `json:"id" gorm:"primaryKey"`
	StudentNo      string        `json:"student_no" gorm:"uniqueIndex;size:20;not null"`
	Name           string        `json:"name" gorm:"size:50;not null"`
	Gender         string        `json:"gender" gorm:"size:10"`
	Age            int           `json:"age"`
	Phone          string        `json:"phone" gorm:"size:20"`
	Email          string        `json:"email" gorm:"size:100"`
	Department     string        `json:"department" gorm:"size:100"`
	Major          string        `json:"major" gorm:"size:100"`
	ClassName      string        `json:"class_name" gorm:"size:50"`
	EnrollmentYear int           `json:"enrollment_year"`
	Status         StudentStatus `json:"status" gorm:"size:20;default:在读"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
